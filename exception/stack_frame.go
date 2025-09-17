package exception

import (
	"runtime"
)

// StackFrame represents a single frame in a stack trace. It contains the
// function name, source file, and line number for a point in the call stack.
type StackFrame struct {
	Function string
	File     string
	Line     int
}

// StackFrames is a slice of StackFrame values. It represents a complete stack
// trace.
type StackFrames []StackFrame

// StackTrace captures the current call stack as StackFrames, starting from the
// caller of StackTrace itself.
//
// The skip parameter controls how many additional stack frames are omitted. A
// value of 0 includes the caller of StackTrace, a value of 1 skips that frame,
// and higher values skip more.
func StackTrace(skip int) StackFrames {
	// get stack trace
	const depth = 64
	var programCounters [depth]uintptr
	programCountersLength := runtime.Callers(2+skip, programCounters[:])
	frames := runtime.CallersFrames(programCounters[:programCountersLength])
	// create stack frames
	stack := make([]StackFrame, 0, programCountersLength)
	for {
		if frame, more := frames.Next(); more {
			stack = append(stack, StackFrame{
				Function: frame.Function,
				File:     frame.File,
				Line:     frame.Line,
			})
		} else {
			break
		}
	}
	return stack
}
