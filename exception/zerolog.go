//go:build !no_zerolog

package exception

import (
	"github.com/rs/zerolog"
)

// MarshalZerologObject marshall this [Exception] as a zerolog object.
func (e String) MarshalZerologObject(event *zerolog.Event) {
	event.Str("error", string(e))
}

func (e fullException) MarshalZerologObject(event *zerolog.Event) {
	event.Str("error", e.Type)
	if e.Message != "" {
		event.Str("message", e.Message)
	}
	switch len(e.Cause) {
	case 0: // skip
	case 1:
		event.AnErr("cause", e.Cause[0])
	default:
		event.Errs("cause", e.Cause)
	}
	switch len(e.Suppressed) {
	case 0: // skip
	case 1:
		event.AnErr("suppressed", e.Suppressed[0])
	default:
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
	switch len(e) {
	case 0: // skip
	case 1:
		event.AnErr("cause", e[0])
	default:
		event.Errs("cause", e)
	}
}

// MarshalZerologObject marshall this [StackFrame] as a zerolog object.
func (f StackFrame) MarshalZerologObject(event *zerolog.Event) {
	event.Str("function", f.Function).Str("file", f.File).Int("line", f.Line)
}

// MarshalZerologArray marshall this [StackFrames] as a zerolog array.
func (s StackFrames) MarshalZerologArray(array *zerolog.Array) {
	for _, frame := range s {
		array.Object(frame)
	}
}
