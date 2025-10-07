package exception

import "fmt"

// Exception defines a lightweight exception model for Go, providing mechanisms
// for chaining causes, tracking suppressed errors, storing recovered values, and
// capturing stack traces.
//
// Methods that return Exception may either modify the current exception in place
// or return a new exception instance. Callers should always use the returned
// value and must not assume that the original exception remains unchanged.
type Exception interface {
	// Error returns a string representation of this Exception in the form of "Type: message"
	Error() string

	// GetType returns the type of this Exception.
	GetType() string

	// GetMessage returns the message of this Exception. Can be empty.
	GetMessage() string

	// SetMessage stores a message inside this Exception.
	//
	// Note: This method may modify the current Exception or return a new one. Always
	// use the returned value.
	SetMessage(message string, parameters ...any) Exception

	// GetCause returns the list of underlying causes associated with this Exception.
	// The slice may be empty if no causes have been specified.
	GetCause() []error

	// AddCause attaches one or more underlying causes to this Exception. Causes are
	// typically used to represent the root errors that led to this exception being
	// raised.
	//
	// Note: This method may modify the current Exception or return a new one. Always
	// use the returned value.
	AddCause(errors ...error) Exception

	// GetSuppressed returns the list of suppressed errors that were intentionally
	// ignored or deferred while handling this Exception. This can be useful when
	// multiple errors occur, but only one is chosen as the primary failure.
	GetSuppressed() []error

	// AddSuppressed attaches one or more suppressed errors to this Exception.
	//
	// Note: This method may modify the current Exception or return a new one. Always
	// use the returned value.
	AddSuppressed(errors ...error) Exception

	// GetRecovered returns the value captured from a panic recovery, if any. It
	// returns nil if no value was recovered.
	GetRecovered() any

	// SetRecovered stores a recovered panic value inside this Exception.
	//
	// Note: This method may modify the current Exception or return a new one. Always
	// use the returned value.
	SetRecovered(recovered any) Exception

	// GetStackTrace returns the stack trace captured for this Exception, represented
	// as a slice of StackFrame values. The slice may be empty if no stack trace was
	// filled.
	GetStackTrace() StackFrames

	// FillStackTrace captures the current call stack starting from the caller of
	// FillStackTrace itself and attaches it to the Exception.
	//
	// The skip parameter controls how many additional stack frames
	// are omitted. A value of 0 includes the caller of FillStackTrace,
	// a value of 1 skips that frame, and higher values skip more.
	//
	// Note: This method may modify the current Exception or return a new one. Always
	// use the returned value.
	FillStackTrace(skip int) Exception

	__() // private
}

type exception struct {
	Type       string
	Message    string
	Cause      []error
	Suppressed []error
	Recovered  any
	StackTrace []StackFrame
}

func (e exception) Error() string {
	if e.Message != "" {
		return e.Type + ":" + e.Message
	}
	return e.Type
}

func (e exception) GetType() string {
	return e.Type
}

func (e exception) GetMessage() string {
	return e.Message
}

func (e exception) SetMessage(message string, parameters ...any) Exception {
	if len(parameters) > 0 {
		e.Message = fmt.Sprintf(message, parameters...)
	} else {
		e.Message = message
	}
	return e
}

func (e exception) GetCause() []error {
	return e.Cause
}

func (e exception) AddCause(errors ...error) Exception {
	concat(&e.Cause, errors...)
	return e
}

func (e exception) GetSuppressed() []error {
	return e.Suppressed
}

func (e exception) AddSuppressed(errors ...error) Exception {
	concat(&e.Suppressed, errors...)
	return e
}

func (e exception) GetRecovered() any {
	return e.Recovered
}

func (e exception) SetRecovered(recovered any) Exception {
	e.Recovered = recovered
	return e
}

func (e exception) GetStackTrace() StackFrames {
	return e.StackTrace
}

func (e exception) FillStackTrace(skip int) Exception {
	e.StackTrace = StackTrace(skip + 1)
	return e
}

func (e exception) __() {}

func (e exception) Unwrap() []error {
	return e.Cause
}

func (e exception) Is(target error) bool {
	return is(e, target)
}

func (e exception) As(target any) bool {
	return as(e, target)
}
