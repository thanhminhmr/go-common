/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package observable

import (
	"sync"
	"sync/atomic"
)

// Observer is called when an Observable emits a matching event.
//
// The event argument is the emitted event name. The args values are the
// arguments passed to Trigger.
//
// Observers are invoked asynchronously by Trigger. Observers should treat args
// as immutable, because the same arguments may be shared by multiple observers.
type Observer func(event string, args ...any)

// Observable is a concurrent, zero-value-usable asynchronous event broadcaster.
//
// Observers can be registered for a specific event name, or for the empty event
// name, which acts as a catch-all subscription.
type Observable struct {
	// fns stores observers by event name.
	//
	// The outer map is map[string]*sync.Map.
	// The inner map is map[uint64]Observer.
	//
	// The empty string key is used for catch-all observers.
	fns sync.Map

	// id generates unique IDs for registered observers.
	id atomic.Uint64
}

// On registers fn as an observer for event.
//
// Use an empty event name to register fn as a catch-all observer. Catch-all
// observers receive every emitted event.
//
// The returned cancel function unregisters fn. It is safe to call cancel
// multiple times.
//
// On panics if fn is nil.
func (o *Observable) On(event string, fn Observer) (cancel func()) {
	if fn == nil {
		panic("observable.On: fn cannot be nil")
	}
	fnsAny, _ := o.fns.LoadOrStore(event, &sync.Map{})
	fns := fnsAny.(*sync.Map)
	id := o.id.Add(1)
	fns.Store(id, fn)
	return sync.OnceFunc(func() { fns.Delete(id) })
}

// Trigger emits event asynchronously.
//
// Each matching observer runs in its own goroutine. Trigger does not wait for
// observers to finish, and observers are not guaranteed to run in registration
// order.
//
// Observers registered for event are called first. If event is non-empty,
// catch-all observers registered for the empty event name are also called.
//
// If observers are added or removed concurrently with Trigger, they may or may
// not observe the in-flight event.
//
// Trigger does not recover panics from observers. If an observer panics, that
// panic occurs in the observer's goroutine and may crash the program unless the
// observer recovers it.
func (o *Observable) Trigger(event string, args ...any) {
	// callback on the event
	o.trigger(event, event, args...)
	// callback on the catch-all event
	if event != "" {
		o.trigger("", event, args...)
	}
}

// trigger invokes observers stored under key for the emitted event.
//
// key is the subscription key to look up in o.fns. event is the event name
// passed to observers. These differ when invoking catch-all observers: key is
// the empty string, while event is the emitted event name.
func (o *Observable) trigger(key, event string, args ...any) {
	if fnsAny, exists := o.fns.Load(key); exists {
		fns := fnsAny.(*sync.Map)
		fns.Range(func(_, fn any) bool {
			go fn.(Observer)(event, args...)
			return true
		})
	}
}
