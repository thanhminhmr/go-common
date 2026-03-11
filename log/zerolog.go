/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

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

const timeFormat = "2006-01-02T15:04:05.000000000Z07:00"

func ConsoleLogger(lifecycle fx.Lifecycle) context.Context {
	// create the logger
	logger := zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: timeFormat,
	}).With().Timestamp().Caller().Logger()
	// create the global context with lifecycle cancel binding and the logger
	ctx, cancel := context.WithCancel(logger.WithContext(context.Background()))
	lifecycle.Append(fx.Hook{
		OnStop: func(context.Context) error {
			cancel()
			return nil
		},
	})
	return ctx
}
