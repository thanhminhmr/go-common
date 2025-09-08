package errors

import (
	"runtime"

	"github.com/rs/zerolog"
)

type StackFrame struct {
	Function string
	File     string
	Line     int
}

func (f StackFrame) MarshalZerologObject(event *zerolog.Event) {
	event.Str("function", f.Function).Str("file", f.File).Int("line", f.Line)
}

type StackFrames []StackFrame

func (s StackFrames) MarshalZerologArray(array *zerolog.Array) {
	for _, frame := range s {
		array.Object(frame)
	}
}

func StackTrace(skip int) StackFrames {
	// get stack trace
	const depth = 32
	var programCounters [depth]uintptr
	programCountersLength := runtime.Callers(2+skip, programCounters[:])
	frames := runtime.CallersFrames(programCounters[:programCountersLength])
	// create stack frames
	var stack [depth]StackFrame
	stackLength := 0
	for ; stackLength < depth; stackLength++ {
		frame, more := frames.Next()
		if !more {
			break
		}
		stack[stackLength] = StackFrame{
			Function: frame.Function,
			File:     frame.File,
			Line:     frame.Line,
		}
	}
	return stack[:stackLength]
}
