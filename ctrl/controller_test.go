/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package ctrl_test

import (
	"context"
	"testing"
	"time"
	_ "unsafe"

	"github.com/thanhminhmr/go-common/ctrl"
	"github.com/thanhminhmr/go-exception"
)

//go:linkname reset github.com/thanhminhmr/go-common/ctrl.reset
func reset()

const errorValue = errorString("custom")

type errorString string

func (e errorString) Error() string {
	return string(e)
}

func initializerPanic() { panic(errorValue) }

func starterPanic(_ context.Context) (ctrl.Runner, ctrl.Cleaner) { panic(errorValue) }

func starterEmpty(_ context.Context) (ctrl.Runner, ctrl.Cleaner) { return nil, nil }

func starterRunnerCleanerPanic(_ context.Context) (ctrl.Runner, ctrl.Cleaner) {
	return runnerPanic, cleanerPanic
}

func starterRunnerShutdown(_ context.Context) (ctrl.Runner, ctrl.Cleaner) {
	return runnerShutdown, nil
}

func runnerPanic(_ context.Context, _ context.CancelFunc) { panic(errorValue) }

func runnerShutdown(_ context.Context, shutdown context.CancelFunc) { shutdown() }

func cleanerPanic(_ context.Context) { panic(errorValue) }

func runnerTimeout(t *testing.T) ctrl.Runner {
	return func(ctx context.Context, shutdown context.CancelFunc) {
		timer := time.After(time.Minute)
		select {
		case <-timer:
			t.Log("timed out")
			t.FailNow()
		case <-ctx.Done():
		}
	}
}

func starterRunnerTimeout(t *testing.T) ctrl.Starter {
	return func(ctx context.Context) (ctrl.Runner, ctrl.Cleaner) {
		return runnerTimeout(t), nil
	}
}

func recoverFailing(t *testing.T) func(exception.Exception) {
	return func(exception.Exception) { t.FailNow() }
}

func recoverLogging(t *testing.T) func(exception.Exception) {
	return func(ex exception.Exception) { t.Log(ex) }
}

func timeoutTestFailed(t *testing.T, fn func()) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	timer := time.After(time.Minute)
	select {
	case <-timer:
		t.Log("timed out")
		t.FailNow()
	case <-done:
	}
}

func starterNotCalledTestFailed(t *testing.T) (ctrl.Starter, func()) {
	runner, runnerCheck := runnerNotCalledTestFailed(t)
	cleaner, cleanerCheck := cleanerNotCalledTestFailed(t)
	called := false
	return func(_ context.Context) (ctrl.Runner, ctrl.Cleaner) {
			called = true
			return runner, cleaner
		}, func() {
			if !called {
				t.FailNow()
			}
			runnerCheck()
			cleanerCheck()
		}
}

func runnerNotCalledTestFailed(t *testing.T) (ctrl.Runner, func()) {
	called := false
	return func(_ context.Context, _ context.CancelFunc) {
			called = true
		}, func() {
			if !called {
				t.FailNow()
			}
		}
}

func cleanerNotCalledTestFailed(t *testing.T) (ctrl.Cleaner, func()) {
	called := false
	return func(_ context.Context) {
			called = true
		}, func() {
			if !called {
				t.FailNow()
			}
		}
}

// ================================================================================

func TestInitializerPanic(t *testing.T) {
	reset()
	timeoutTestFailed(t, func() {
		defer exception.Recover(recoverFailing(t))
		ctrl.Control(initializerPanic)
	})
}

func TestStrayRegister(t *testing.T) {
	reset()
	timeoutTestFailed(t, func() {
		defer exception.Recover(recoverLogging(t))
		ctrl.Register(starterEmpty)
		t.FailNow() // stray Register should panic
	})
}

func TestRegisterPanic(t *testing.T) {
	reset()
	timeoutTestFailed(t, func() {
		defer exception.Recover(recoverLogging(t))
		ctrl.Control(func() {
			ctrl.Register(starterPanic)
			t.FailNow()
		})
	})
}

func TestRunnerCleanerPanic(t *testing.T) {
	reset()
	starter, deferCheck := starterNotCalledTestFailed(t)
	defer deferCheck()
	timeoutTestFailed(t, func() {
		defer exception.Recover(recoverFailing(t))
		ctrl.Control(func() {
			defer exception.Recover(recoverFailing(t))
			ctrl.Register(starterRunnerCleanerPanic)
			ctrl.Register(starter)
			ctrl.Register(starterRunnerTimeout(t))
		})
	})
}

func TestNominal(t *testing.T) {
	reset()
	starter, deferCheck := starterNotCalledTestFailed(t)
	defer deferCheck()
	timeoutTestFailed(t, func() {
		defer exception.Recover(recoverFailing(t))
		ctrl.Control(func() {
			defer exception.Recover(recoverFailing(t))
			ctrl.Register(starter)
		})
	})
}

func TestShutdown(t *testing.T) {
	reset()
	starter, deferCheck := starterNotCalledTestFailed(t)
	defer deferCheck()
	timeoutTestFailed(t, func() {
		defer exception.Recover(recoverFailing(t))
		ctrl.Control(func() {
			defer exception.Recover(recoverFailing(t))
			ctrl.Register(starterRunnerTimeout(t))
			ctrl.Register(starterRunnerShutdown)
			ctrl.Register(starter)
		})
	})
}
