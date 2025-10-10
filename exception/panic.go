package exception

// PanicError is the default type for exceptions created by [Panic] and
// [Recover].
const PanicError = String("panicked")

// Panic behaves like the built-in panic, but always panics with an [Exception]
// whose type is [PanicError].
//
// If the value already implements [Exception] and its type is [PanicError], it
// is re-panicked directly. This allows Panic to be used in a chain of recover
// handlers without changing the original panic state.
//
// Otherwise, [Panic] creates a new [Exception] that uses [PanicError] as its
// type, keeps the recovered value, and records the stack trace starting from the
// caller.
//
// Typical usage together with [Recover]:
//
//	defer func() {
//	    if err := exception.Recover(recover()); err != nil {
//	        // handle exception
//	    }
//	}()
//
//	...
//
//	if somethingWrong {
//	    exception.Panic("bad state")
//	}
func Panic(recovered any) {
	if err, ok := recovered.(Exception); !ok || err.GetType() != string(PanicError) {
		recovered = fullException{
			Type:       string(PanicError),
			Recovered:  recovered,
			StackTrace: StackTrace(1),
		}
	}
	panic(recovered)
}

// Recover normalizes a recovered panic value into an [Exception].
//
// If recovered is nil, [Recover] returns nil.
//
// If the value already implements [Exception] and its type is [PanicError], it
// is returned directly. This allows multiple recover handlers to work together:
// the first one captures the stack trace, and later ones can observe or rethrow
// the same [Exception] without modification.
//
// Otherwise, [Recover] creates a new [Exception] that uses [PanicError] as its
// type, keeps the recovered value, and records the stack trace starting from the
// location where the panic occurred.
//
// Typical usage in a deferred function:
//
//	defer func() {
//	    if err := exception.Recover(recover()); err != nil {
//	        // handle exception
//	    }
//	}()
func Recover(recovered any) Exception {
	if recovered == nil {
		return nil
	}
	if err, ok := recovered.(Exception); ok && err.GetType() == string(PanicError) {
		return err
	}
	// skip to panic frame if exists
	trace := StackTrace(1)
	for i, frame := range trace {
		if frame.Function == "runtime.gopanic" {
			trace = trace[i+1:]
			break
		}
	}
	return fullException{
		Type:       string(PanicError),
		Recovered:  recovered,
		StackTrace: trace,
	}
}
