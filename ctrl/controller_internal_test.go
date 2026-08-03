/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package ctrl

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// ResetForTest resets the controller state machine back to uninitialized.
func ResetForTest() { controller = controllerState{} }

func testLoggerCtx() context.Context {
	return zerolog.Ctx(context.Background()).WithContext(context.Background())
}

func TestCleanTimeout_CleanerCompletes(t *testing.T) {
	called := false
	cleaner := func(_ context.Context) { called = true }
	cleanTimeout(testLoggerCtx(), cleaner, 5*time.Second)
	assert.True(t, called, "cleaner was not called")
}

func TestCleanTimeout_CleanerExceedsTimeout(t *testing.T) {
	done := make(chan struct{})
	cleaner := func(ctx context.Context) {
		<-ctx.Done()
		close(done)
	}
	cleanTimeout(testLoggerCtx(), cleaner, 50*time.Millisecond)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("cleaner ctx was not canceled by timeout")
	}
}
