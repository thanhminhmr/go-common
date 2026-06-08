/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package ctrl

import (
	"context"
	"fmt"
	"net"
	"time"
	"unsafe"

	"github.com/rs/zerolog"
)

type LogLevel = zerolog.Level
type LogEvent = zerolog.Event
type LogArray = zerolog.Array
type LogArrayMarshaler = zerolog.LogArrayMarshaler
type LogObjectMarshaler = zerolog.LogObjectMarshaler

func LogCtx(ctx context.Context) LogContext {
	if ctx == nil {
		return LogContext{logger: globalLogger}.setCtx(context.Background())
	}
	if ctx, ok := ctx.(LogContext); ok {
		return ctx
	}
	if value := ctx.Value(&logWriter); value != nil {
		return LogContext{logger: value.(zerolog.Logger)}.setCtx(ctx)
	}
	return LogContext{logger: globalLogger}.setCtx(ctx)
}

type LogContext struct {
	logger zerolog.Logger
}

type zerologLogger struct {
	w       zerolog.LevelWriter
	level   zerolog.Level
	sampler zerolog.Sampler
	context []byte
	hooks   []zerolog.Hook
	stack   bool
	ctx     context.Context
}

func (c LogContext) ctx() context.Context {
	return (*zerologLogger)(unsafe.Pointer(&c.logger)).ctx
}

func (c LogContext) setCtx(ctx context.Context) LogContext {
	(*zerologLogger)(unsafe.Pointer(&c.logger)).ctx = ctx
	return c
}

func (c LogContext) Deadline() (deadline time.Time, ok bool) {
	return c.ctx().Deadline()
}

func (c LogContext) Done() <-chan struct{} {
	return c.ctx().Done()
}

func (c LogContext) Err() error {
	return c.ctx().Err()
}

func (c LogContext) Value(key any) any {
	if key == &logWriter {
		return c.logger
	}
	return c.ctx().Value(key)
}

func (c LogContext) WithValue(key, value any) LogContext {
	ctx := context.WithValue(c.ctx(), key, value)
	return c.setCtx(ctx)
}

func (c LogContext) WithCancel() (LogContext, context.CancelFunc) {
	ctx, cancel := context.WithCancel(c.ctx())
	return c.setCtx(ctx), cancel
}

func (c LogContext) WithCancelCause() (LogContext, context.CancelCauseFunc) {
	ctx, cancel := context.WithCancelCause(c.ctx())
	return c.setCtx(ctx), cancel
}

func (c LogContext) WithTimeout(timeout time.Duration) (LogContext, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(c.ctx(), timeout)
	return c.setCtx(ctx), cancel
}

func (c LogContext) WithTimeoutCause(timeout time.Duration, cause error) (LogContext, context.CancelFunc) {
	ctx, cancel := context.WithTimeoutCause(c.ctx(), timeout, cause)
	return c.setCtx(ctx), cancel
}

func (c LogContext) WithDeadline(deadline time.Time) (LogContext, context.CancelFunc) {
	ctx, cancel := context.WithDeadline(c.ctx(), deadline)
	return c.setCtx(ctx), cancel
}

func (c LogContext) WithDeadlineCause(deadline time.Time, cause error) (LogContext, context.CancelFunc) {
	ctx, cancel := context.WithDeadlineCause(c.ctx(), deadline, cause)
	return c.setCtx(ctx), cancel
}

func (c LogContext) With() LogBuilder {
	return LogBuilder{ctx: c.logger.With()}
}

func (c LogContext) Trace() *LogEvent { return c.logger.Trace() }

func (c LogContext) Debug() *LogEvent { return c.logger.Debug() }

func (c LogContext) Info() *LogEvent { return c.logger.Info() }

func (c LogContext) Warn() *LogEvent { return c.logger.Warn() }

func (c LogContext) Error() *LogEvent { return c.logger.Error() }

func (c LogContext) Level(level LogLevel) *LogEvent { return c.logger.WithLevel(level) }

type LogBuilder struct {
	ctx zerolog.Context
}

// Logger returns the logger with the context previously set.
func (b LogBuilder) Logger() LogContext {
	return LogContext{logger: b.ctx.Logger()}
}

func (b LogBuilder) Fields(value any) LogBuilder {
	return LogBuilder{ctx: b.ctx.Fields(value)}
}

func (b LogBuilder) Dict(key string, value *LogEvent) LogBuilder {
	return LogBuilder{ctx: b.ctx.Dict(key, value)}
}

func (b LogBuilder) CreateDict() *LogEvent {
	return b.ctx.CreateDict()
}

func (b LogBuilder) CreateArray() *LogArray {
	return b.ctx.CreateArray()
}

func (b LogBuilder) Array(key string, value LogArrayMarshaler) LogBuilder {
	return LogBuilder{ctx: b.ctx.Array(key, value)}
}

func (b LogBuilder) Object(key string, value LogObjectMarshaler) LogBuilder {
	return LogBuilder{ctx: b.ctx.Object(key, value)}
}

func (b LogBuilder) Objects(key string, values []LogObjectMarshaler) LogBuilder {
	return LogBuilder{ctx: b.ctx.Objects(key, values)}
}

func (b LogBuilder) ObjectsV(key string, values ...LogObjectMarshaler) LogBuilder {
	return LogBuilder{ctx: b.ctx.ObjectsV(key, values...)}
}

func (b LogBuilder) EmbedObject(value LogObjectMarshaler) LogBuilder {
	return LogBuilder{ctx: b.ctx.EmbedObject(value)}
}

func (b LogBuilder) Str(key string, value string) LogBuilder {
	return LogBuilder{ctx: b.ctx.Str(key, value)}
}

func (b LogBuilder) Strs(key string, values []string) LogBuilder {
	return LogBuilder{ctx: b.ctx.Strs(key, values)}
}

func (b LogBuilder) StrsV(key string, values ...string) LogBuilder {
	return LogBuilder{ctx: b.ctx.StrsV(key, values...)}
}

func (b LogBuilder) Stringer(key string, value fmt.Stringer) LogBuilder {
	return LogBuilder{ctx: b.ctx.Stringer(key, value)}
}

func (b LogBuilder) Stringers(key string, values []fmt.Stringer) LogBuilder {
	return LogBuilder{ctx: b.ctx.Stringers(key, values)}
}

func (b LogBuilder) StringersV(key string, values ...fmt.Stringer) LogBuilder {
	return LogBuilder{ctx: b.ctx.StringersV(key, values...)}
}

func (b LogBuilder) Bytes(key string, values []byte) LogBuilder {
	return LogBuilder{ctx: b.ctx.Bytes(key, values)}
}

func (b LogBuilder) Hex(key string, values []byte) LogBuilder {
	return LogBuilder{ctx: b.ctx.Hex(key, values)}
}

func (b LogBuilder) RawJSON(key string, values []byte) LogBuilder {
	return LogBuilder{ctx: b.ctx.RawJSON(key, values)}
}

func (b LogBuilder) AnErr(key string, value error) LogBuilder {
	return LogBuilder{ctx: b.ctx.AnErr(key, value)}
}

func (b LogBuilder) Errs(key string, values []error) LogBuilder {
	return LogBuilder{ctx: b.ctx.Errs(key, values)}
}

func (b LogBuilder) Err(value error) LogBuilder {
	return LogBuilder{ctx: b.ctx.Err(value)}
}

func (b LogBuilder) Ctx(value context.Context) LogBuilder {
	return LogBuilder{ctx: b.ctx.Ctx(value)}
}

func (b LogBuilder) Bool(key string, value bool) LogBuilder {
	return LogBuilder{ctx: b.ctx.Bool(key, value)}
}

func (b LogBuilder) Bools(key string, values []bool) LogBuilder {
	return LogBuilder{ctx: b.ctx.Bools(key, values)}
}

func (b LogBuilder) Int(key string, value int) LogBuilder {
	return LogBuilder{ctx: b.ctx.Int(key, value)}
}

func (b LogBuilder) Ints(key string, values []int) LogBuilder {
	return LogBuilder{ctx: b.ctx.Ints(key, values)}
}

func (b LogBuilder) Int8(key string, value int8) LogBuilder {
	return LogBuilder{ctx: b.ctx.Int8(key, value)}
}

func (b LogBuilder) Ints8(key string, values []int8) LogBuilder {
	return LogBuilder{ctx: b.ctx.Ints8(key, values)}
}

func (b LogBuilder) Int16(key string, value int16) LogBuilder {
	return LogBuilder{ctx: b.ctx.Int16(key, value)}
}

func (b LogBuilder) Ints16(key string, values []int16) LogBuilder {
	return LogBuilder{ctx: b.ctx.Ints16(key, values)}
}

func (b LogBuilder) Int32(key string, value int32) LogBuilder {
	return LogBuilder{ctx: b.ctx.Int32(key, value)}
}

func (b LogBuilder) Ints32(key string, values []int32) LogBuilder {
	return LogBuilder{ctx: b.ctx.Ints32(key, values)}
}

func (b LogBuilder) Int64(key string, value int64) LogBuilder {
	return LogBuilder{ctx: b.ctx.Int64(key, value)}
}

func (b LogBuilder) Ints64(key string, values []int64) LogBuilder {
	return LogBuilder{ctx: b.ctx.Ints64(key, values)}
}

func (b LogBuilder) Uint(key string, value uint) LogBuilder {
	return LogBuilder{ctx: b.ctx.Uint(key, value)}
}

func (b LogBuilder) Uints(key string, values []uint) LogBuilder {
	return LogBuilder{ctx: b.ctx.Uints(key, values)}
}

func (b LogBuilder) Uint8(key string, value uint8) LogBuilder {
	return LogBuilder{ctx: b.ctx.Uint8(key, value)}
}

func (b LogBuilder) Uints8(key string, values []uint8) LogBuilder {
	return LogBuilder{ctx: b.ctx.Uints8(key, values)}
}

func (b LogBuilder) Uint16(key string, value uint16) LogBuilder {
	return LogBuilder{ctx: b.ctx.Uint16(key, value)}
}

func (b LogBuilder) Uints16(key string, values []uint16) LogBuilder {
	return LogBuilder{ctx: b.ctx.Uints16(key, values)}
}

func (b LogBuilder) Uint32(key string, value uint32) LogBuilder {
	return LogBuilder{ctx: b.ctx.Uint32(key, value)}
}

func (b LogBuilder) Uints32(key string, values []uint32) LogBuilder {
	return LogBuilder{ctx: b.ctx.Uints32(key, values)}
}

func (b LogBuilder) Uint64(key string, value uint64) LogBuilder {
	return LogBuilder{ctx: b.ctx.Uint64(key, value)}
}

func (b LogBuilder) Uints64(key string, values []uint64) LogBuilder {
	return LogBuilder{ctx: b.ctx.Uints64(key, values)}
}

func (b LogBuilder) Float32(key string, value float32) LogBuilder {
	return LogBuilder{ctx: b.ctx.Float32(key, value)}
}

func (b LogBuilder) Floats32(key string, values []float32) LogBuilder {
	return LogBuilder{ctx: b.ctx.Floats32(key, values)}
}

func (b LogBuilder) Float64(key string, value float64) LogBuilder {
	return LogBuilder{ctx: b.ctx.Float64(key, value)}
}

func (b LogBuilder) Floats64(key string, values []float64) LogBuilder {
	return LogBuilder{ctx: b.ctx.Floats64(key, values)}
}

func (b LogBuilder) Timestamp() LogBuilder {
	return LogBuilder{ctx: b.ctx.Timestamp()}
}

func (b LogBuilder) Time(key string, value time.Time) LogBuilder {
	return LogBuilder{ctx: b.ctx.Time(key, value)}
}

func (b LogBuilder) Times(key string, values []time.Time) LogBuilder {
	return LogBuilder{ctx: b.ctx.Times(key, values)}
}

func (b LogBuilder) Dur(key string, value time.Duration) LogBuilder {
	return LogBuilder{ctx: b.ctx.Dur(key, value)}
}

func (b LogBuilder) Durs(key string, values []time.Duration) LogBuilder {
	return LogBuilder{ctx: b.ctx.Durs(key, values)}
}

func (b LogBuilder) Interface(key string, value any) LogBuilder {
	return LogBuilder{ctx: b.ctx.Interface(key, value)}
}

func (b LogBuilder) Type(key string, value any) LogBuilder {
	return LogBuilder{ctx: b.ctx.Type(key, value)}
}

func (b LogBuilder) Any(key string, value any) LogBuilder {
	return LogBuilder{ctx: b.ctx.Any(key, value)}
}

func (b LogBuilder) Reset() LogBuilder {
	return LogBuilder{ctx: b.ctx.Reset()}
}

func (b LogBuilder) Caller() LogBuilder {
	return LogBuilder{ctx: b.ctx.Caller()}
}

func (b LogBuilder) CallerWithSkipFrameCount(value int) LogBuilder {
	return LogBuilder{ctx: b.ctx.CallerWithSkipFrameCount(value)}
}

func (b LogBuilder) Stack() LogBuilder {
	return LogBuilder{ctx: b.ctx.Stack()}
}

func (b LogBuilder) IPAddr(key string, value net.IP) LogBuilder {
	return LogBuilder{ctx: b.ctx.IPAddr(key, value)}
}

func (b LogBuilder) IPAddrs(key string, values []net.IP) LogBuilder {
	return LogBuilder{ctx: b.ctx.IPAddrs(key, values)}
}

func (b LogBuilder) IPPrefix(key string, value net.IPNet) LogBuilder {
	return LogBuilder{ctx: b.ctx.IPPrefix(key, value)}
}

func (b LogBuilder) IPPrefixes(key string, values []net.IPNet) LogBuilder {
	return LogBuilder{ctx: b.ctx.IPPrefixes(key, values)}
}

func (b LogBuilder) MACAddr(key string, value net.HardwareAddr) LogBuilder {
	return LogBuilder{ctx: b.ctx.MACAddr(key, value)}
}
