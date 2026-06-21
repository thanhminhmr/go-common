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

func Register(starter Starter) {
	RegisterWithTimeout(starter, time.Duration(config.ControllerStarterTimeout)*time.Second)
}

func RegisterWithTimeout(starter Starter, timeout time.Duration) {
	if starter == nil {
		panic("BUG: starter is nil")
	}
	if status.Load() != statusIsInitializing {
		panic("BUG: Controller state is unexpected")
	}
	startTimeout(Logger(globalCtx), starter, timeout)
}

func Control(initializer Initializer) {
	if initializer == nil {
		panic("BUG: Initializer is nil")
	}
	// set the state to initializing
	if !status.CompareAndSwap(statusIsUninitialized, statusIsInitializing) {
		panic("BUG: Controller state is unexpected")
	}
	// defer shutdown cleanly
	defer func() {
		shutdown()
		<-cleaned
	}()
	// register a recover
	defer exception.Recover(func(recovered exception.Exception) {
		Logger(globalCtx).Error().AnErr("recovered", recovered).Msg("Initializer panicked")
	})
	// run initializer
	logCtx := Logger(globalCtx)
	logCtx.Info().Msg("Initializing...")
	initializer()
	logCtx.Info().Msg("Initialized, starting runners...")
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
	logCtx.Info().Msg("Runners started")
	wait.Wait()
	logCtx.Info().Msg("Runners finished")
}

func GlobalCtx() context.Context {
	return globalCtx
}

var (
	globalCtx context.Context
	shutdown  context.CancelFunc
	cleaners  []Cleaner
	cleaned   chan struct{}
	runners   []Runner
	status    atomic.Uintptr
	wait      sync.WaitGroup
	mutex     sync.Mutex
)

const (
	statusIsUninitialized = iota
	statusIsInitializing
	statusIsRunning
	statusIsTerminating
)

func setupController(logCleaner Cleaner) {
	// create global context
	globalCtx, shutdown = context.WithCancel(context.Background())
	// set cleaners to have log cleaner as a default
	cleaners = []Cleaner{logCleaner}
	// create cleaned channel
	cleaned = make(chan struct{})
	// clear runners
	runners = nil
	// reset status
	status = atomic.Uintptr{}
	// reset wait group
	wait = sync.WaitGroup{}
	// reset mutex
	mutex = sync.Mutex{}
	// create cleaning handler
	context.AfterFunc(globalCtx, cleanAll)
}

func cleanAll() {
	defer close(cleaned)
	status.Store(statusIsTerminating)
	logger := Logger(context.Background())
	logger.Info().Msg("Cleaning up...")
	defer logger.Info().Msg("Cleaned up")
	timeout := time.Duration(config.ControllerCleanerTimeout) * time.Second
	for _, cleaner := range slices.Backward(cleaners) {
		cleanTimeout(logger, cleaner, timeout)
	}
}

func cleanTimeout(logCtx *LogCtx, cleaner Cleaner, timeout time.Duration) {
	// timeout
	if timeout > 0 {
		var cancel context.CancelFunc
		logCtx, cancel = logCtx.WithTimeout(timeout)
		defer cancel()
	}
	// run and wait
	done := make(chan struct{})
	go cleanOne(logCtx, cleaner, done)
	select {
	case <-logCtx.Done():
		logCtx.Warn().Err(logCtx.Err()).Msg("Cleaner cancelled")
	case <-done:
	}
}

func cleanOne(logCtx *LogCtx, cleaner Cleaner, done chan<- struct{}) {
	defer close(done)
	defer exception.Recover(func(recovered exception.Exception) {
		Logger(globalCtx).Error().AnErr("recovered", recovered).Msg("Cleaner panicked")
	})
	cleaner(logCtx)
}

func startTimeout(logCtx *LogCtx, starter Starter, timeout time.Duration) {
	// lock
	mutex.Lock()
	defer mutex.Unlock()
	// timeout
	if timeout > 0 {
		var cancel context.CancelFunc
		logCtx, cancel = logCtx.WithTimeout(timeout)
		defer cancel()
	}
	// run and wait
	done := make(chan exception.Exception)
	go startOne(logCtx, starter, done)
	select {
	case <-logCtx.Done():
		err := logCtx.Err()
		logCtx.Error().Err(err).Msg("Starter cancelled")
		panic(err)
	case recovered, exists := <-done:
		if exists {
			panic(recovered)
		}
	}
}

func startOne(logCtx *LogCtx, starter Starter, done chan<- exception.Exception) {
	defer close(done)
	defer exception.Recover(func(recovered exception.Exception) { done <- recovered })
	// call starter
	runner, cleaner := starter(logCtx)
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
	defer exception.Recover(func(recovered exception.Exception) {
		Logger(globalCtx).Error().AnErr("recovered", recovered).Msg("Runner panicked")
		shutdown()
	})
	runner(globalCtx, shutdown)
}
