/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package ctrl

import (
	"github.com/thanhminhmr/go-common/configuration"
)

type Config struct {
	ControllerStarterTimeout uint     `env:"CONTROLLER_CONTROLLER_STARTER_TIMEOUT" default:"30"`
	ControllerCleanerTimeout uint     `env:"CONTROLLER_CONTROLLER_CLEANER_TIMEOUT" default:"30"`
	LoggerUnixTimestamp      bool     `env:"LOGGER_UNIX_TIMESTAMP"`
	LoggerMinimumLevel       LogLevel `env:"LOGGER_MINIMUM_LEVEL" default:"trace" validate:"min=-1,max=7"`
}

var config Config

func init() {
	// load config
	if err := configuration.LoadInto(&config); err != nil {
		panic(err)
	}
	// run setup
	setup()
}

func setup() {
	// setup logger
	setupLogger()
	// set up the controller
	setupController(logWriter.syncMode)
}
