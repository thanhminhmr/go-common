package log

import (
	"context"
	"log/slog"
	"os"
	"reflect"
	"time"

	"github.com/thanhminhmr/go-common/configuration"

	"github.com/phuslu/log"
)

func init() {
	config, err := configuration.Load[Config]()
	if err != nil {
		panic(err)
	}
	slog.SetDefault(slog.New(log.SlogNewJSONHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: config.AddSource,
		Level:     config.Level,
	})))
}

type Config struct {
	AddSource bool       `env:"LOGGER_ADD_SOURCE"`
	Level     slog.Level `env:"LOGGER_LEVEL" default:"DEBUG"`
}

func IntoCtx(ctx context.Context) *Ctx {
	if ctx, ok := ctx.(*Ctx); ok {
		return ctx
	}
	if value := ctx.Value(reflect.TypeFor[*slog.Logger]()); value != nil {
		if logger, ok := value.(*slog.Logger); ok {
			return &Ctx{
				ctx:    ctx,
				logger: logger,
			}
		}
	}
	return &Ctx{
		ctx:    ctx,
		logger: slog.Default(),
	}
}

type Ctx struct {
	ctx    context.Context
	logger *slog.Logger
}

func (c *Ctx) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}

func (c *Ctx) Done() <-chan struct{} {
	return c.ctx.Done()
}

func (c *Ctx) Err() error {
	return c.ctx.Err()
}

func (c *Ctx) Value(key any) any {
	if key == reflect.TypeFor[*slog.Logger]() {
		return c.logger
	}
	return c.ctx.Value(key)
}

func (c *Ctx) With(args ...any) *Ctx {
	return &Ctx{
		ctx:    c.ctx,
		logger: c.logger.With(args...),
	}
}

func (c *Ctx) WithGroup(name string) *Ctx {
	return &Ctx{
		ctx:    c.ctx,
		logger: c.logger.WithGroup(name),
	}
}

func (c *Ctx) Enabled(level slog.Level) bool {
	return c.logger.Enabled(c.ctx, level)
}

func (c *Ctx) Log(level slog.Level, msg string, args ...any) {
	c.logger.Log(c.ctx, level, msg, args...)
}

func (c *Ctx) LogAttrs(level slog.Level, msg string, attrs ...slog.Attr) {
	c.logger.LogAttrs(c.ctx, level, msg, attrs...)
}

func (c *Ctx) Debug(msg string, args ...any) {
	c.logger.DebugContext(c.ctx, msg, args...)
}

func (c *Ctx) Info(msg string, args ...any) {
	c.logger.InfoContext(c.ctx, msg, args...)
}

func (c *Ctx) Warn(msg string, args ...any) {
	c.logger.WarnContext(c.ctx, msg, args...)
}

func (c *Ctx) Error(msg string, args ...any) {
	c.logger.ErrorContext(c.ctx, msg, args...)
}
