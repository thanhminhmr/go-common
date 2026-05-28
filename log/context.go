/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package log

import (
	"context"
	"log/slog"
	"reflect"
	"runtime"
	"sync"
	"time"
)

func Logger(ctx context.Context) Ctx {
	if ctx == nil {
		return Ctx{
			ctx:     context.Background(),
			handler: handler,
		}
	}
	if ctx, ok := ctx.(Ctx); ok {
		return ctx
	}
	if value := ctx.Value(reflect.TypeFor[private]()); value != nil {
		return Ctx{
			ctx:     ctx,
			handler: value.(unsafeHandler),
		}
	}
	return Ctx{
		ctx:     ctx,
		handler: handler,
	}
}

type Ctx struct {
	ctx     context.Context
	handler unsafeHandler
}

func (c Ctx) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}

func (c Ctx) Done() <-chan struct{} {
	return c.ctx.Done()
}

func (c Ctx) Err() error {
	return c.ctx.Err()
}

func (c Ctx) Value(key any) any {
	if key == reflect.TypeFor[private]() {
		return c.handler
	}
	return c.ctx.Value(key)
}

func (c Ctx) WithValue(key, value any) Ctx {
	return Ctx{ctx: context.WithValue(c.ctx, key, value), handler: c.handler}
}

func (c Ctx) WithCancel() (Ctx, context.CancelFunc) {
	ctx, cancel := context.WithCancel(c.ctx)
	return Ctx{ctx: ctx, handler: c.handler}, cancel
}

func (c Ctx) WithCancelCause() (Ctx, context.CancelCauseFunc) {
	ctx, cancel := context.WithCancelCause(c.ctx)
	return Ctx{ctx: ctx, handler: c.handler}, cancel
}

func (c Ctx) WithTimeout(timeout time.Duration) (Ctx, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(c.ctx, timeout)
	return Ctx{ctx: ctx, handler: c.handler}, cancel
}

func (c Ctx) WithTimeoutCause(timeout time.Duration, cause error) (Ctx, context.CancelFunc) {
	ctx, cancel := context.WithTimeoutCause(c.ctx, timeout, cause)
	return Ctx{ctx: ctx, handler: c.handler}, cancel
}

func (c Ctx) WithDeadline(deadline time.Time) (Ctx, context.CancelFunc) {
	ctx, cancel := context.WithDeadline(c.ctx, deadline)
	return Ctx{ctx: ctx, handler: c.handler}, cancel
}

func (c Ctx) WithDeadlineCause(deadline time.Time, cause error) (Ctx, context.CancelFunc) {
	ctx, cancel := context.WithDeadlineCause(c.ctx, deadline, cause)
	return Ctx{ctx: ctx, handler: c.handler}, cancel
}

func (c Ctx) With(args ...any) Ctx {
	return Ctx{
		ctx:     c.ctx,
		handler: slog.New(c.handler).With(args...).Handler().(unsafeHandler),
	}
}

func (c Ctx) WithGroup(name string) Ctx {
	return Ctx{
		ctx:     c.ctx,
		handler: c.handler.WithGroup(name).(unsafeHandler),
	}
}

func (c Ctx) log(level slog.Level) *Entry {
	if !c.handler.Enabled(c.ctx, level) {
		return nil
	}
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])
	record := recordPool.Get().(*Entry)
	*record = Entry{
		rec: slog.Record{
			Level: level,
			PC:    pcs[0],
		},
		log: c,
	}
	return record
}

func (c Ctx) Log(level slog.Level) *Entry {
	return c.log(level)
}

func (c Ctx) Debug() *Entry {
	return c.log(slog.LevelDebug)
}

func (c Ctx) Info() *Entry {
	return c.log(slog.LevelInfo)
}

func (c Ctx) Warn() *Entry {
	return c.log(slog.LevelWarn)
}

func (c Ctx) Error() *Entry {
	return c.log(slog.LevelError)
}

var recordPool = sync.Pool{New: func() any { return new(Entry) }}

type Entry struct {
	rec slog.Record
	log Ctx
}

func (r *Entry) With(args ...any) *Entry {
	if r == nil {
		return nil
	}
	r.rec.Add(args...)
	return r
}

func (r *Entry) Msg(msg string) {
	if r == nil {
		return
	}
	r.rec.Time = time.Now()
	r.rec.Message = msg
	_ = r.log.handler.Handle(r.log, r.rec)
	*r = Entry{}
	recordPool.Put(r)
	return
}
