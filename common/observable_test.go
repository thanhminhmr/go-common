/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package common_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thanhminhmr/go-common/common"
)

// waitRecv waits for a signal on ch or fails the test after a timeout.
func waitRecv(t *testing.T, ch <-chan struct{}, msg string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for: %s", msg)
	}
}

// assertNoRecv fails if a signal arrives on ch within the window. Used for
// negative assertions on observers that must not fire. The window only needs to
// outlast scheduler jitter: when an observer is genuinely unregistered, no
// goroutine is ever launched for it, so this is not timing-sensitive.
func assertNoRecv(t *testing.T, ch <-chan struct{}, msg string) {
	t.Helper()
	select {
	case <-ch:
		t.Fatalf("unexpected: %s", msg)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestObservableZeroValue(t *testing.T) {
	// A zero-value Observable is immediately usable without initialization.
	var o common.Observable
	called := make(chan struct{}, 1)
	o.On("e", func(event string, args ...any) { called <- struct{}{} })
	o.Trigger("e")
	waitRecv(t, called, "zero-value observer call")
}

func TestObservableOnTriggerEventAndArgs(t *testing.T) {
	var o common.Observable
	got := make(chan [3]any, 1)
	o.On("click", func(event string, args ...any) {
		got <- [3]any{event, args[0], args[1]}
	})
	o.Trigger("click", 42, "hello")
	select {
	case v := <-got:
		if v[0] != "click" || v[1] != 42 || v[2] != "hello" {
			t.Errorf("got %v, want [click 42 hello]", v)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: observer not called")
	}
}

func TestObservableMultipleObserversSameEvent(t *testing.T) {
	var o common.Observable
	const n = 5
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		o.On("e", func(event string, args ...any) { wg.Done() })
	}
	o.Trigger("e")
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: not all observers called")
	}
}

func TestObservableDifferentEvents(t *testing.T) {
	var o common.Observable
	aCalled := make(chan struct{}, 1)
	bCalled := make(chan struct{}, 1)
	o.On("a", func(event string, args ...any) { aCalled <- struct{}{} })
	o.On("b", func(event string, args ...any) { bCalled <- struct{}{} })
	o.Trigger("a")
	waitRecv(t, aCalled, "observer a should fire")
	assertNoRecv(t, bCalled, "observer b should not fire on event a")
}

func TestObservableCatchAllReceivesEventAndArgs(t *testing.T) {
	var o common.Observable
	got := make(chan [2]any, 1)
	o.On("", func(event string, args ...any) { got <- [2]any{event, args[0]} })
	o.Trigger("hello", 7)
	select {
	case v := <-got:
		if v[0] != "hello" || v[1] != 7 {
			t.Errorf("catch-all got %v, want [hello 7]", v)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: catch-all not called")
	}
}

func TestObservableCatchAllFiresOnceForEmptyEvent(t *testing.T) {
	// Trigger("") must invoke the catch-all exactly once (the empty-event
	// guard in Trigger prevents a second dispatch).
	var o common.Observable
	got := make(chan struct{}, 4)
	o.On("", func(event string, args ...any) { got <- struct{}{} })
	o.Trigger("")
	waitRecv(t, got, "catch-all for empty event")
	assertNoRecv(t, got, "catch-all called twice for empty event")
}

func TestObservableSpecificAndCatchAllBothFire(t *testing.T) {
	var o common.Observable
	specific := make(chan string, 1)
	catchAll := make(chan string, 1)
	o.On("e", func(event string, args ...any) { specific <- event })
	o.On("", func(event string, args ...any) { catchAll <- event })
	o.Trigger("e", "arg")
	select {
	case ev := <-specific:
		if ev != "e" {
			t.Errorf("specific got %q, want %q", ev, "e")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: specific observer not called")
	}
	select {
	case ev := <-catchAll:
		if ev != "e" {
			t.Errorf("catch-all got %q, want %q", ev, "e")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: catch-all observer not called")
	}
}

func TestObservableCancelUnregisters(t *testing.T) {
	var o common.Observable
	called := make(chan struct{}, 1)
	cancel := o.On("e", func(event string, args ...any) { called <- struct{}{} })
	cancel()
	o.Trigger("e")
	assertNoRecv(t, called, "observer called after cancel")
}

func TestObservableCancelIdempotent(t *testing.T) {
	var o common.Observable
	cancel := o.On("e", func(event string, args ...any) {})
	cancel()
	cancel() // must not panic
	cancel()
}

func TestObservableCancelDoesNotAffectOtherObservers(t *testing.T) {
	var o common.Observable
	keepCalled := make(chan struct{}, 1)
	cancel := o.On("e", func(event string, args ...any) {})
	o.On("e", func(event string, args ...any) { keepCalled <- struct{}{} })
	cancel()
	o.Trigger("e")
	waitRecv(t, keepCalled, "other observer should still fire after one is cancelled")
}

func TestObservableMultipleTriggers(t *testing.T) {
	var o common.Observable
	const n = 10
	var wg sync.WaitGroup
	wg.Add(n)
	o.On("e", func(event string, args ...any) { wg.Done() })
	for i := 0; i < n; i++ {
		o.Trigger("e")
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: not all triggers delivered")
	}
}

func TestObservableTriggerNoObservers(t *testing.T) {
	// Triggering events with no registered observers must be a safe no-op.
	var o common.Observable
	o.Trigger("nobody")
	o.Trigger("")
}

func TestObservableOnPanicsOnNil(t *testing.T) {
	var o common.Observable
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when registering nil observer")
		}
	}()
	o.On("e", nil)
}

func TestObservableConcurrent(t *testing.T) {
	// Each goroutine uses a unique event name so triggers only invoke their
	// own observer; this isolates the call count while still stressing the
	// shared Observable under concurrent On/Trigger/Cancel.
	var o common.Observable
	const n = 100
	var wg sync.WaitGroup
	var count atomic.Int64
	for i := 0; i < n; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			ev := fmt.Sprintf("e%d", i)
			done := make(chan struct{})
			cancel := o.On(ev, func(event string, args ...any) {
				count.Add(1)
				close(done)
			})
			o.Trigger(ev, i)
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Errorf("observer %d not called", i)
			}
			cancel()
		}()
	}
	wg.Wait()
	if c := count.Load(); c != n {
		t.Errorf("got %d calls, want %d", c, n)
	}
}

func TestObservableCancelConcurrentWithTrigger(t *testing.T) {
	// Observers added or removed concurrently with Trigger may or may not
	// observe the in-flight event, so the call count is intentionally
	// nondeterministic. This test exists to catch data races under
	// `go test -race`.
	var o common.Observable
	const n = 200
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cancel := o.On("e", func(event string, args ...any) {})
			o.Trigger("e")
			cancel()
		}()
	}
	wg.Wait()
}
