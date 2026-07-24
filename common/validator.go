/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package common

import "github.com/go-playground/validator/v10"

var globalValidator = validator.New(validator.WithRequiredStructEnabled())

func ValidateStruct(value any) error { return globalValidator.Struct(value) }
