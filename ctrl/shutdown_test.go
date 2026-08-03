/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package ctrl_test

import (
	"context"
	"io"
	"os/signal"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thanhminhmr/go-common/ctrl"
)

func TestShutdownOnSignal_SignalReceived(t *testing.T) {
	t.Cleanup(func() { signal.Reset(syscall.SIGINT, syscall.SIGTERM) })

	loggerCtx := zerolog.New(io.Discard).WithContext(context.Background())
	runner, _ := ctrl.ShutdownOnSignal(loggerCtx, loggerCtx)

	ctx, cancel := context.WithCancel(loggerCtx)
	t.Cleanup(cancel)

	var shutdownCalled atomic.Bool
	done := make(chan struct{})
	go func() {
		defer close(done)
		runner(ctx, func() { shutdownCalled.Store(true) })
	}()

	// Give the runner time to register signal.Notify.
	time.Sleep(100 * time.Millisecond)
	require.NoError(t, syscall.Kill(syscall.Getpid(), syscall.SIGTERM))

	select {
	case <-done:
	case <-time.After(testTimeout):
		t.Fatal("runner did not return after signal")
	}
	assert.True(t, shutdownCalled.Load(), "shutdown was not called after signal")
}

func TestShutdownOnSignal_ContextCanceled(t *testing.T) {
	loggerCtx := zerolog.New(io.Discard).WithContext(context.Background())
	runner, _ := ctrl.ShutdownOnSignal(loggerCtx, loggerCtx)

	ctx, cancel := context.WithCancel(loggerCtx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		runner(ctx, func() {})
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(testTimeout):
		t.Fatal("runner did not return after ctx cancel")
	}
}
