package exception

import "fmt"

// type check
var _ Exception = String("")

// String is a string-based [Exception]. It behaves like a simple error
// containing only a message, with no causes, suppressed errors, recovered value,
// or stack trace.
//
// [String] is often used as a starting point for building a full exception with
// additional context. When causes, suppressed errors, or stack traces are added,
// a new [Exception] will be created that keeps the message and includes the
// added details:
//
//	err := exception.String("read failed").FillStackTrace(0)
//
// [String] can also be used as a constant error value, for example:
//
//	const ErrRead = exception.String("read failed")
type String string

// Error returns a string representation of this exception in the form of "Type:
// Message"
func (e String) Error() string {
	return string(e)
}

// GetType returns the type of this exception.
func (e String) GetType() string {
	return string(e)
}

// GetMessage returns the message of this exception.
func (e String) GetMessage() string {
	return ""
}

// SetMessage stores a message inside this exception.
//
// Note: This method may modify the current exception or return a new one. Always
// use the returned [Exception].
func (e String) SetMessage(message string, parameters ...any) Exception {
	if message == "" {
		return e
	}
	if len(parameters) > 0 {
		return fullException{
			Type:    string(e),
			Message: fmt.Sprintf(message, parameters...),
		}
	}
	return fullException{
		Type:    string(e),
		Message: message,
	}
}

// GetCause returns the list of underlying causes associated with this exception.
// The slice may be empty if no causes have been specified.
func (e String) GetCause() []error {
	return nil
}

// AddCause attaches one or more underlying causes to this exception. Causes are
// typically used to represent the root errors that led to this exception being
// raised.
//
// Note: This method may modify the current exception or return a new one. Always
// use the returned [Exception].
func (e String) AddCause(errors ...error) Exception {
	var cause []error
	if combine(&cause, errors...) {
		return fullException{
			Type:  string(e),
			Cause: cause,
		}
	}
	return e
}

// GetSuppressed returns the list of suppressed errors that were intentionally
// ignored or deferred while handling this exception. This can be useful when
// multiple errors occur, but only one is chosen as the primary failure.
func (e String) GetSuppressed() []error {
	return nil
}

// AddSuppressed attaches one or more suppressed errors to this exception.
//
// Note: This method may modify the current exception or return a new one. Always
// use the returned [Exception].
func (e String) AddSuppressed(errors ...error) Exception {
	var suppressed []error
	if combine(&suppressed, errors...) {
		return fullException{
			Type:       string(e),
			Suppressed: suppressed,
		}
	}
	return e
}

// GetRecovered returns the value captured from a panic recovery, if any. It
// returns nil if no value was recovered.
func (e String) GetRecovered() any {
	return nil
}

// SetRecovered stores a recovered panic value inside this exception.
//
// Note: This method may modify the current exception or return a new one. Always
// use the returned [Exception].
func (e String) SetRecovered(recovered any) Exception {
	if recovered == nil {
		return e
	}
	return fullException{
		Type:      string(e),
		Recovered: recovered,
	}
}

// GetStackTrace returns the stack trace captured for this exception, represented
// as [StackFrames]. The result may be nil if no stack trace was filled.
func (e String) GetStackTrace() StackFrames {
	return nil
}

// FillStackTrace captures the current call stack starting from the caller of
// [FillStackTrace] itself and attaches it to this exception.
//
// The skip parameter controls how many additional stack frames are omitted. A
// value of 0 includes the caller of [FillStackTrace], a value of 1 skips that
// frame, and higher values skip more.
//
// Note: This method may modify the current exception or return a new one. Always
// use the returned [Exception].
func (e String) FillStackTrace(skip int) Exception {
	return fullException{
		Type:       string(e),
		StackTrace: StackTrace(skip + 1),
	}
}

func (e String) __() {}

func (e String) Is(target error) bool {
	return is(e, target)
}

func (e String) As(target any) bool {
	return as(e, target)
}
