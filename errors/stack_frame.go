package errors

import (
	"runtime"
)

type StackFrame struct {
	Function string
	File     string
	Line     int
}

type StackFrames []StackFrame

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
