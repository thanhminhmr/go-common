/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package ctrl_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/thanhminhmr/go-common/ctrl"
)

const testTimeout = 5 * time.Second

var panicSentinel = errors.New("test panic")

// runWithTimeout runs fn in a goroutine and fails the test if it does not
// complete within testTimeout. Any panic escaping fn is reported as an error.
func runWithTimeout(t *testing.T, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("unexpected panic: %v", r)
			}
		}()
		fn()
	}()
	select {
	case <-time.After(testTimeout):
		t.Fatal("test timed out")
	case <-done:
	}
}

// blockingRunner returns a Runner that blocks until ctx is canceled.
func blockingRunner() ctrl.Runner {
	return func(ctx context.Context, _ context.CancelFunc) { <-ctx.Done() }
}

// shutdownRunner returns a Runner that immediately calls shutdown.
func shutdownRunner() ctrl.Runner {
	return func(_ context.Context, shutdown context.CancelFunc) { shutdown() }
}

// ================================================================================
// Control: nil and state-machine panic cases
// ================================================================================

func TestControl_NilInitializer_Panics(t *testing.T) {
	ctrl.ResetForTest()
	assert.Panics(t, func() { ctrl.Control(nil) })
}

func TestControl_SecondCall_Panics(t *testing.T) {
	ctrl.ResetForTest()
	runWithTimeout(t, func() {
		ctrl.Control(func(_ *zerolog.Logger) {
			ctrl.Register(func(_, _ context.Context) (ctrl.Runner, ctrl.Cleaner) {
				return shutdownRunner(), nil
			})
		})
	})
	// After Control returns, status is terminating (set by cleanAll), so the
	// CAS uninitialized→initializing fails and Control panics.
	assert.Panics(t, func() { ctrl.Control(func(_ *zerolog.Logger) {}) })
}

// ================================================================================
// Control: panic recovery
// ================================================================================

func TestControl_InitializerPanic_Recovered(t *testing.T) {
	ctrl.ResetForTest()
	runWithTimeout(t, func() {
		ctrl.Control(func(_ *zerolog.Logger) { panic(panicSentinel) })
	})
}

func TestControl_StarterPanic_Recovered(t *testing.T) {
	ctrl.ResetForTest()
	runWithTimeout(t, func() {
		ctrl.Control(func(_ *zerolog.Logger) {
			ctrl.Register(func(_, _ context.Context) (ctrl.Runner, ctrl.Cleaner) {
				panic(panicSentinel)
			})
		})
	})
}

func TestControl_StarterTimeout_Recovered(t *testing.T) {
	ctrl.ResetForTest()
	release := make(chan struct{})
	starterReturned := make(chan struct{})
	runWithTimeout(t, func() {
		ctrl.Control(func(_ *zerolog.Logger) {
			ctrl.RegisterWithTimeout(func(ctx, _ context.Context) (ctrl.Runner, ctrl.Cleaner) {
				<-ctx.Done()
				<-release // keep the starter goroutine alive past the panic
				close(starterReturned)
				return nil, nil
			}, 50*time.Millisecond)
		})
	})
	// Release the starter and wait for it (and thus startOne's read of
	// controller.globalCtx at controller.go:228) to finish before returning,
	// establishing happens-before against the next test's ResetForTest.
	close(release)
	select {
	case <-starterReturned:
	case <-time.After(testTimeout):
		t.Fatal("starter did not return after release")
	}
}

func TestControl_RunnerCleanerPanic_Recovered(t *testing.T) {
	ctrl.ResetForTest()
	var starterCalled, runnerCalled, cleanerCalled atomic.Bool
	runWithTimeout(t, func() {
		ctrl.Control(func(_ *zerolog.Logger) {
			ctrl.Register(func(_, _ context.Context) (ctrl.Runner, ctrl.Cleaner) {
				return func(_ context.Context, _ context.CancelFunc) { panic(panicSentinel) },
					func(_ context.Context) { panic(panicSentinel) }
			})
			ctrl.Register(func(_, _ context.Context) (ctrl.Runner, ctrl.Cleaner) {
				starterCalled.Store(true)
				return func(ctx context.Context, _ context.CancelFunc) {
					runnerCalled.Store(true)
					<-ctx.Done()
				}, func(_ context.Context) { cleanerCalled.Store(true) }
			})
		})
	})
	assert.True(t, starterCalled.Load(), "second starter not called")
	assert.True(t, runnerCalled.Load(), "second runner not called")
	assert.True(t, cleanerCalled.Load(), "second cleaner not called")
}

// ================================================================================
// Control: nominal flow
// ================================================================================

func TestControl_Nominal(t *testing.T) {
	ctrl.ResetForTest()
	var starterCalled, runnerCalled, cleanerCalled atomic.Bool
	runWithTimeout(t, func() {
		ctrl.Control(func(_ *zerolog.Logger) {
			ctrl.Register(func(_, _ context.Context) (ctrl.Runner, ctrl.Cleaner) {
				starterCalled.Store(true)
				return func(ctx context.Context, shutdown context.CancelFunc) {
					runnerCalled.Store(true)
					shutdown()
					<-ctx.Done()
				}, func(_ context.Context) { cleanerCalled.Store(true) }
			})
		})
	})
	assert.True(t, starterCalled.Load(), "starter not called")
	assert.True(t, runnerCalled.Load(), "runner not called")
	assert.True(t, cleanerCalled.Load(), "cleaner not called")
}

func TestControl_Shutdown_StopsRunners(t *testing.T) {
	ctrl.ResetForTest()
	var blockingDone atomic.Bool
	runWithTimeout(t, func() {
		ctrl.Control(func(_ *zerolog.Logger) {
			ctrl.Register(func(_, _ context.Context) (ctrl.Runner, ctrl.Cleaner) {
				return shutdownRunner(), nil
			})
			ctrl.Register(func(_, _ context.Context) (ctrl.Runner, ctrl.Cleaner) {
				return func(ctx context.Context, _ context.CancelFunc) {
					<-ctx.Done()
					blockingDone.Store(true)
				}, nil
			})
		})
	})
	assert.True(t, blockingDone.Load(), "blocking runner was not stopped by shutdown")
}

func TestControl_RunnerReceivesGlobalCtx(t *testing.T) {
	ctrl.ResetForTest()
	runWithTimeout(t, func() {
		ctrl.Control(func(_ *zerolog.Logger) {
			ctrl.Register(func(_, globalCtx context.Context) (ctrl.Runner, ctrl.Cleaner) {
				return func(ctx context.Context, shutdown context.CancelFunc) {
					assert.Equal(t, globalCtx, ctx, "runner ctx is not the globalCtx")
					shutdown()
					<-globalCtx.Done()
				}, nil
			})
		})
	})
}

func TestControl_CleanersRunInReverseOrder(t *testing.T) {
	ctrl.ResetForTest()
	var mu sync.Mutex
	order := make([]int, 0, 3)
	runWithTimeout(t, func() {
		ctrl.Control(func(_ *zerolog.Logger) {
			for i := range 3 {
				ctrl.Register(func(_, _ context.Context) (ctrl.Runner, ctrl.Cleaner) {
					return shutdownRunner(), func(_ context.Context) {
						mu.Lock()
						order = append(order, i)
						mu.Unlock()
					}
				})
			}
		})
	})
	assert.Equal(t, []int{2, 1, 0}, order)
}

// ================================================================================
// Register / RegisterWithTimeout
// ================================================================================

func TestRegister_OutsideInitializer_Panics(t *testing.T) {
	ctrl.ResetForTest()
	assert.Panics(t, func() {
		ctrl.Register(func(_, _ context.Context) (ctrl.Runner, ctrl.Cleaner) { return nil, nil })
	})
}

func TestRegister_NilStarter_Panics(t *testing.T) {
	ctrl.ResetForTest()
	assert.Panics(t, func() { ctrl.Register(nil) })
}

func TestRegisterWithTimeout_Zero_NoDeadline(t *testing.T) {
	ctrl.ResetForTest()
	var starterCalled atomic.Bool
	runWithTimeout(t, func() {
		ctrl.Control(func(_ *zerolog.Logger) {
			ctrl.RegisterWithTimeout(func(_, _ context.Context) (ctrl.Runner, ctrl.Cleaner) {
				starterCalled.Store(true)
				return shutdownRunner(), nil
			}, 0)
		})
	})
	assert.True(t, starterCalled.Load(), "starter not called")
}
