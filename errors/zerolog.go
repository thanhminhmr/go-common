//go:build !no_zerolog

package errors

import "github.com/rs/zerolog"

func (e String) MarshalZerologObject(event *zerolog.Event) {
	event.Str("error", string(e))
}

func (e fullError) MarshalZerologObject(event *zerolog.Event) {
	event.Str("error", e.String)
	if e.Cause != nil {
		event.Errs("cause", e.Cause)
	}
	if e.Suppressed != nil {
		event.Errs("suppressed", e.Suppressed)
	}
	if e.Recovered != nil {
		event.Any("recovered", e.Recovered)
	}
	if e.StackTrace != nil {
		event.Any("stack_trace", e.StackTrace)
	}
}

func (e multipleErrors) MarshalZerologObject(event *zerolog.Event) {
	event.Errs("cause", e)
}

func (f StackFrame) MarshalZerologObject(event *zerolog.Event) {
	event.Str("function", f.Function).Str("file", f.File).Int("line", f.Line)
}

func (s StackFrames) MarshalZerologArray(array *zerolog.Array) {
	for _, frame := range s {
		array.Object(frame)
	}
}
