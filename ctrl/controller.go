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

	"github.com/thanhminhmr/go-common/cfg"

	"github.com/rs/zerolog"
	"github.com/thanhminhmr/go-exception"
)

// Config configures the controller state machine. It is loaded from the
// "controller" section of the config file via [cfg.LoadInto] during [Control].
//
//   - StarterTimeout bounds each [Starter] call. Zero disables the deadline.
//   - CleanerTimeout bounds each [Cleaner] call during shutdown. Zero disables the
//     deadline.
type Config struct {
	StarterTimeout uint `cfg:"starter_timeout" default:"30"`
	CleanerTimeout uint `cfg:"cleaner_timeout" default:"30"`
}

// Initializer initializes the controller state machine by calling [Register] as
// needed, and panic if any error occurred in the process. Anything that had been
// registered before the panic will be clean up gracefully.
type Initializer = func(globalCtx context.Context)

// Starter starts the service, usually with a timeout. The ctx parameter is the
// starter's deadline context, bounded by [Config.StarterTimeout] (or the
// timeout passed to [RegisterWithTimeout]); the starter must return before ctx
// is canceled or the controller panics. The globalCtx parameter is the
// controller's global context, shared across all services and canceled when
// [Control] shuts down. A Starter that spawns long-lived work should bind
// globalCtx into its [Runner] and [Cleaner] so they observe shutdown. It
// returns a [Runner] and a [Cleaner]; either may be nil if not needed.
type Starter = func(ctx, globalCtx context.Context) (Runner, Cleaner)

// Runner runs a service. It is returned by a [Starter] and launched as a
// goroutine by [Control]. The runner should run until ctx is canceled (the
// controller calls shutdown to cancel it) or until it calls shutdown itself to
// initiate a graceful shutdown. If a Runner panics, the controller recovers it,
// logs the error, and initiates shutdown.
type Runner = func(ctx context.Context, shutdown context.CancelFunc)

// Cleaner releases resources held by a service. It is returned by a [Starter]
// and run during shutdown in reverse registration order, with a per-cleaner
// timeout governed by [Config.CleanerTimeout].
type Cleaner = func(ctx context.Context)

// Register registers a [Starter] with the controller, using [Config.StarterTimeout]
// as the deadline. It must be called from inside the [Initializer] passed to
// [Control]; calling it outside that context panics.
func Register(starter Starter) {
	RegisterWithTimeout(starter, time.Duration(controller.config.StarterTimeout)*time.Second)
}

// RegisterWithTimeout is like [Register] but with an explicit starter timeout.
// A timeout of 0 disables the deadline. Must be called from inside the
// [Initializer] passed to [Control].
func RegisterWithTimeout(starter Starter, timeout time.Duration) {
	if starter == nil {
		panic("BUG: starter is nil")
	}
	if controller.status.Load() != statusIsInitializing {
		panic("BUG: Controller state is unexpected")
	}
	startTimeout(controller.globalCtx, starter, timeout)
}

// Control runs the controller state machine. It loads config, runs the
// initializer (which calls [Register] for each service), then launches all
// registered runners as goroutines and blocks until they finish. On return the
// controller runs all registered cleaners in reverse order and waits for them
// to complete. If the initializer panics, the controller recovers it, logs the
// error, and still runs any cleaners registered before the panic.
//
// Control panics if called when the controller is not in the uninitialized
// state (e.g. a second call, or a [Register] outside an initializer).
func Control(initializer Initializer) {
	if initializer == nil {
		panic("BUG: Initializer is nil")
	}
	// set the state to initializing
	if !controller.status.CompareAndSwap(statusIsUninitialized, statusIsInitializing) {
		panic("BUG: Controller state is unexpected")
	}
	// load controller config
	if err := cfg.LoadInto(&controller.config, "controller"); err != nil {
		panic(err)
	}
	// create global context
	controller.globalCtx, controller.shutdown = context.WithCancel(context.Background())
	// create cleaning handler
	controller.cleaned = make(chan struct{})
	// register cleaning callback
	context.AfterFunc(controller.globalCtx, cleanAll)
	// defer shutdown cleanly
	defer func() {
		controller.shutdown()
		<-controller.cleaned
	}()
	// register a recover
	logger := zerolog.Ctx(controller.globalCtx)
	defer exception.Recover(func(recovered exception.Exception) {
		logger.Error().AnErr("recovered", recovered).Msg("Initializer panicked")
	})
	// run initializer
	logger.Info().Msg("Initializing...")
	initializer(controller.globalCtx)
	logger.Info().Msg("Initialized, starting runners...")
	// set the state to running
	if !controller.status.CompareAndSwap(statusIsInitializing, statusIsRunning) {
		panic("BUG: Controller state is unexpected")
	}
	// start runners
	for _, runner := range controller.runners {
		// start the runner
		controller.wait.Add(1)
		go runOne(runner)
	}
	// wait for all runner/clean up to finish
	logger.Info().Msg("Runners started")
	controller.wait.Wait()
	logger.Info().Msg("Runners finished")
}

type controllerState = struct {
	config    Config
	status    atomic.Uintptr
	globalCtx context.Context
	shutdown  context.CancelFunc
	runners   []Runner
	cleaners  []Cleaner
	cleaned   chan struct{}
	wait      sync.WaitGroup
	mutex     sync.Mutex
}

// controller is the global state machine's mutable state.
var controller controllerState

const (
	statusIsUninitialized = iota
	statusIsInitializing
	statusIsRunning
	statusIsTerminating
)

func cleanAll() {
	defer close(controller.cleaned)
	controller.status.Store(statusIsTerminating)
	ctx := context.Background()
	logger := zerolog.Ctx(ctx)
	logger.Info().Msg("Cleaning up...")
	defer logger.Info().Msg("Cleaned up")
	timeout := time.Duration(controller.config.CleanerTimeout) * time.Second
	for _, cleaner := range slices.Backward(controller.cleaners) {
		cleanTimeout(ctx, cleaner, timeout)
	}
}

func cleanTimeout(ctx context.Context, cleaner Cleaner, timeout time.Duration) {
	// timeout
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	// run and wait
	done := make(chan struct{})
	go cleanOne(ctx, cleaner, done)
	select {
	case <-ctx.Done():
		zerolog.Ctx(ctx).Warn().Err(ctx.Err()).Msg("Cleaner cancelled")
	case <-done:
	}
}

func cleanOne(ctx context.Context, cleaner Cleaner, done chan<- struct{}) {
	defer close(done)
	defer exception.Recover(func(recovered exception.Exception) {
		zerolog.Ctx(ctx).Error().AnErr("recovered", recovered).Msg("Cleaner panicked")
	})
	cleaner(ctx)
}

func startTimeout(ctx context.Context, starter Starter, timeout time.Duration) {
	// lock
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	// timeout
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	// run and wait
	done := make(chan exception.Exception)
	go startOne(ctx, starter, done)
	select {
	case <-ctx.Done():
		err := ctx.Err()
		zerolog.Ctx(ctx).Error().Err(err).Msg("Starter cancelled")
		panic(err)
	case recovered, exists := <-done:
		if exists {
			panic(recovered)
		}
	}
}

func startOne(ctx context.Context, starter Starter, done chan<- exception.Exception) {
	defer close(done)
	defer exception.Recover(func(recovered exception.Exception) { done <- recovered })
	// call starter
	runner, cleaner := starter(ctx, controller.globalCtx)
	// add runner/cleaner if any
	if runner != nil {
		controller.runners = append(controller.runners, runner)
	}
	if cleaner != nil {
		controller.cleaners = append(controller.cleaners, cleaner)
	}
}

func runOne(runner Runner) {
	defer controller.wait.Done()
	defer exception.Recover(func(recovered exception.Exception) {
		zerolog.Ctx(controller.globalCtx).Error().AnErr("recovered", recovered).Msg("Runner panicked")
		controller.shutdown()
	})
	runner(controller.globalCtx, controller.shutdown)
}
