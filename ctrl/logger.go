/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package ctrl

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/thanhminhmr/go-common/cfg"
)

// loggerConfig configures the zerolog global logger setup.
type loggerConfig struct {
	UnixTimestamp bool          `cfg:"unix_timestamp"`
	MinimumLevel  zerolog.Level `cfg:"minimum_level" default:"trace" validate:"min=-1,max=7"`
}

func init() {
	config, err := cfg.Load[loggerConfig]("logger")
	if err != nil {
		panic(err)
	}
	zerolog.SetGlobalLevel(config.MinimumLevel)
	if config.UnixTimestamp {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnixNano
	} else {
		zerolog.TimeFieldFormat = time.RFC3339Nano
	}
	globalLogger := zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()
	zerolog.DefaultContextLogger = &globalLogger
}
