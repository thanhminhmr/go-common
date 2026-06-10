/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package ctrl

import (
	"context"
	"unsafe"

	"github.com/rs/zerolog"
)

type zerologLogger struct {
	w       zerolog.LevelWriter
	level   zerolog.Level
	sampler zerolog.Sampler
	context []byte
	hooks   []zerolog.Hook
	stack   bool
	ctx     context.Context
}

var _ = [1]any{unsafe.Sizeof(zerologLogger{}) - unsafe.Sizeof(zerolog.Logger{}): 0}

type zerologLoggerContext interface {
	zerolog.Logger | zerolog.Context
}

func getCtx[T zerologLoggerContext](logger *T) context.Context {
	return (*zerologLogger)(unsafe.Pointer(logger)).ctx
}
