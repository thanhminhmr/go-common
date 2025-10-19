package log

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixNano
}

func ConsoleLogger(lifecycle fx.Lifecycle) (*zerolog.Logger, context.Context) {
	// create the logger
	logger := zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "2006-01-02T15:04:05.000000000Z07:00",
	}).With().Timestamp().Caller().Logger()
	// create the global context with lifecycle cancel binding and the logger
	ctx, cancel := context.WithCancel(logger.WithContext(context.Background()))
	lifecycle.Append(fx.Hook{
		OnStop: func(context.Context) error {
			cancel()
			return nil
		},
	})
	return zerolog.Ctx(ctx), ctx
}
