/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package log

import (
	"reflect"
	"runtime"

	"github.com/rs/zerolog"
	"github.com/thanhminhmr/go-exception"
)

func toStackFrame(v any) exception.StackFrame {
	if v == nil {
		return exception.StackFrame{
			Function: "<nil>",
			File:     "",
			Line:     0,
		}
	}
	r := reflect.ValueOf(v)
	for r.Kind() == reflect.Ptr || r.Kind() == reflect.Interface {
		r = r.Elem()
	}
	switch r.Kind() {
	case reflect.Func:
		f := runtime.FuncForPC(r.Pointer())
		if f == nil {
			return exception.StackFrame{
				Function: "<unknown>",
				File:     "",
				Line:     0,
			}
		}
		file, line := f.FileLine(f.Entry())
		return exception.StackFrame{
			Function: f.Name(),
			File:     file,
			Line:     line,
		}
	default:
		return exception.StackFrame{
			Function: "<invalid>",
			File:     "",
			Line:     0,
		}
	}
}

func FuncOrAny(e *zerolog.Event, k string, v any) *zerolog.Event {
	if v != nil {
		m := toStackFrame(v)
		if m.Function != "<invalid>" {
			return e.Object(k, m)
		}
	}
	return e.Any(k, v)
}

func Func(v any) zerolog.LogObjectMarshaler {
	return toStackFrame(v)
}

func Funcs[S ~[]E, E any](values S) zerolog.LogArrayMarshaler {
	frames := make(exception.StackFrames, 0, len(values))
	for _, value := range values {
		frames = append(frames, toStackFrame(value))
	}
	return frames
}
