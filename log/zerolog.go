/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package log

import (
	"os"

	"github.com/rs/zerolog"
)

type Config struct {
	TimestampFormat     string `env:"LOGGER_TIMESTAMP_FORMAT" validator:"required" default:"2006-01-02T15:04:05.999999999Z07:00"`
	TimestampResolution string `env:"LOGGER_TIMESTAMP_RESOLUTION" validator:"oneof=seconds milliseconds microseconds nanoseconds" default:"nanoseconds"`
}

func ConsoleLogger(config *Config) *zerolog.Logger {
	// config global resolution
	switch config.TimestampResolution {
	case "seconds":
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	case "milliseconds":
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	case "microseconds":
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMicro
	case "nanoseconds":
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnixNano
	}
	// create the logger
	logger := zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: config.TimestampFormat,
	}).With().Timestamp().Caller().Logger()
	return &logger
}
