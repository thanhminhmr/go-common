/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package log

import (
	"context"
	"log/slog"
	"unsafe"
)

type private struct{ *private }

type unsafeHandler struct {
	p unsafe.Pointer
}

func (u unsafeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	h := (*slog.JSONHandler)(unsafe.Pointer(&private{private: (*private)(u.p)}))
	return h.Enabled(ctx, level)
}

func (u unsafeHandler) Handle(ctx context.Context, record slog.Record) error {
	h := (*slog.JSONHandler)(unsafe.Pointer(&private{private: (*private)(u.p)}))
	return h.Handle(ctx, record)
}

func (u unsafeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h := (*slog.JSONHandler)(unsafe.Pointer(&private{private: (*private)(u.p)}))
	return innerHandler(h.WithAttrs(attrs).(*slog.JSONHandler))
}

func (u unsafeHandler) WithGroup(name string) slog.Handler {
	h := (*slog.JSONHandler)(unsafe.Pointer(&private{private: (*private)(u.p)}))
	return innerHandler(h.WithGroup(name).(*slog.JSONHandler))
}

func innerHandler(h *slog.JSONHandler) unsafeHandler {
	return unsafeHandler{p: unsafe.Pointer((*private)(unsafe.Pointer(h)).private)}
}
