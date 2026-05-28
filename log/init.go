/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package log

import (
	"log/slog"
	"os"

	"github.com/thanhminhmr/go-common/configuration"
)

type Config struct {
	AddSource bool       `env:"LOGGER_ADD_SOURCE" default:"true"`
	Level     slog.Level `env:"LOGGER_LEVEL" default:"DEBUG"`
}

var handler unsafeHandler

func init() {
	// load config
	config, err := configuration.Load[Config]()
	if err != nil {
		panic(err)
	}
	// create logger
	handler = innerHandler(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: config.AddSource,
		Level:     config.Level,
	}))
	// set default logger
	slog.SetDefault(slog.New(handler))
}
