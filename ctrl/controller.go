/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package ctrl

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thanhminhmr/go-common/configuration"
	"github.com/thanhminhmr/go-common/log"
	"github.com/thanhminhmr/go-exception"
)

// Initializer initializes the controller state machine by calling [Register] as
// needed, and panic if any error occurred in the process. Anything that had been
// registered before the panic will be clean up gracefully.
type Initializer = func()

// Starter starts the service, usually with a timeout.
type Starter = func(ctx context.Context) (Runner, Cleaner)

type Runner = func(ctx context.Context, shutdown context.CancelFunc)
type Cleaner = func(ctx context.Context)

type Config struct {
	StarterTimeout uint `env:"CONTROLLER_STARTER_TIMEOUT" default:"30"`
	CleanerTimeout uint `env:"CONTROLLER_CLEANER_TIMEOUT" default:"30"`
}

var (
	config    Config
	globalCtx context.Context
	shutdown  context.CancelFunc
	cleaned   chan struct{}
	status    atomic.Uintptr
	wait      sync.WaitGroup
	mutex     sync.Mutex
	runners   []Runner
	cleaners  []Cleaner
)

const (
	statusIsUninitialized = iota
	statusIsInitializing
	statusIsRunning
	statusIsTerminating
)

func init() {
	set()
}

func set() {
	// load config
	if err := configuration.LoadInto(&config); err != nil {
		panic(err)
	}
	// create global context
	globalCtx, shutdown = context.WithCancel(context.Background())
	// create cleaned channel
	cleaned = make(chan struct{})
	// create cleaning handler
	context.AfterFunc(globalCtx, cleanAll)
}

//goland:noinspection GoUnusedFunction
func reset() {
	set()
	status = atomic.Uintptr{}
	wait = sync.WaitGroup{}
	runners = nil
	cleaners = nil
}

func cleanAll() {
	defer close(cleaned)
	status.Store(statusIsTerminating)
	logger := log.Logger(context.Background())
	timeout := time.Duration(config.CleanerTimeout) * time.Second
	for _, cleaner := range slices.Backward(cleaners) {
		cleanTimeout(logger, cleaner, timeout)
	}
}

func cleanTimeout(logger log.Ctx, cleaner Cleaner, timeout time.Duration) {
	if timeout > 0 {
		var cancel context.CancelFunc
		logger, cancel = logger.WithTimeout(timeout)
		defer cancel()
		done := make(chan struct{})
		go cleanOne(logger, cleaner, done)
		select {
		case <-logger.Done():
			logger.Warn().With("error", logger.Err()).Msg("Cleaner timeout")
		case <-done:
		}
	} else {
		cleanOne(logger, cleaner, nil)
	}
}

func cleanOne(logger log.Ctx, cleaner Cleaner, done chan<- struct{}) {
	if done != nil {
		defer close(done)
	}
	defer exception.Recover(func(recovered exception.Exception) {
		logger.Error().With("recovered", recovered).Msg("Cleaner panicked")
	})
	cleaner(logger)
}

func startTimeout(logger log.Ctx, starter Starter, timeout time.Duration) {
	if timeout > 0 {
		var cancel context.CancelFunc
		logger, cancel = logger.WithTimeout(timeout)
		defer cancel()
		done := make(chan any)
		go startOne(logger, starter, done)
		select {
		case <-logger.Done():
			logger.Warn().With("error", logger.Err()).Msg("Starter timeout")
		case recovered := <-done:
			if recovered != nil {
				panic(recovered)
			}
		}
	} else {
		startOne(logger, starter, nil)
	}
}

func startOne(logger log.Ctx, starter Starter, done chan<- any) {
	if done != nil {
		defer close(done)
		defer exception.Recover(func(recovered exception.Exception) { done <- recovered })
	}
	// lock
	mutex.Lock()
	defer mutex.Unlock()
	// call starter
	runner, cleaner := starter(logger)
	// add runner/cleaner if any
	if runner != nil {
		runners = append(runners, runner)
	}
	if cleaner != nil {
		cleaners = append(cleaners, cleaner)
	}
}

func runOne(runner Runner) {
	defer wait.Done()
	logger := log.Logger(globalCtx)
	defer exception.Recover(func(recovered exception.Exception) {
		logger.Error().With("recovered", recovered).Msg("Runner panicked")
	})
	runner(logger, shutdown)
}

func Register(starter Starter) {
	RegisterWithTimeout(starter, time.Duration(config.StarterTimeout)*time.Second)
}

func RegisterWithTimeout(starter Starter, timeout time.Duration) {
	if starter == nil {
		panic("BUG: starter is nil")
	}
	if status.Load() != statusIsInitializing {
		panic("BUG: Controller state is unexpected")
	}
	startTimeout(log.Logger(globalCtx), starter, timeout)
}

func Run(initialize Initializer) {
	if initialize == nil {
		panic("BUG: Initializer is nil")
	}
	// set the state to initializing
	if !status.CompareAndSwap(statusIsUninitialized, statusIsInitializing) {
		panic("BUG: Controller state is unexpected")
	}
	// defer shutdown cleanly
	logger := log.Logger(globalCtx)
	defer logger.Info().Msg("Cleaned up")
	defer func() {
		shutdown()
		<-cleaned
	}()
	// register a recover
	defer exception.Recover(func(recovered exception.Exception) {
		logger.Error().With("recovered", recovered).Msg("Initializer panicked")
	})
	// run initializer
	logger.Info().Msg("Initializing...")
	initialize()
	logger.Info().Msg("Initialized, starting runners...")
	// set the state to running
	if !status.CompareAndSwap(statusIsInitializing, statusIsRunning) {
		panic("BUG: Controller state is unexpected")
	}
	// start runners
	for _, runner := range runners {
		// start the runner
		wait.Add(1)
		go runOne(runner)
	}
	// wait for all runner/clean up to finish
	logger.Info().Msg("Runners started")
	wait.Wait()
	logger.Info().Msg("Runners finished, cleaning up...")
}
