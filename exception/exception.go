package exception

// Exception defines a lightweight exception model for Go, providing mechanisms
// for chaining causes, tracking suppressed errors, storing recovered values, and
// capturing stack traces.
//
// Methods that return [Exception] may either modify the current exception in
// place or return a new exception instance. Callers should always use the
// returned value and must not assume that the original exception remains
// unchanged.
type Exception interface {
	// Error returns a string representation of this exception in the form of "Type:
	// Message"
	Error() string

	// GetType returns the type of this exception.
	GetType() string

	// GetMessage returns the message of this exception.
	GetMessage() string

	// SetMessage stores a message inside this exception.
	//
	// Note: This method may modify the current exception or return a new one. Always
	// use the returned [Exception].
	SetMessage(message string, parameters ...any) Exception

	// GetCause returns the list of underlying causes associated with this exception.
	// The slice may be empty if no causes have been specified.
	GetCause() []error

	// AddCause attaches one or more underlying causes to this exception. Causes are
	// typically used to represent the root errors that led to this exception being
	// raised.
	//
	// Note: This method may modify the current exception or return a new one. Always
	// use the returned [Exception].
	AddCause(errors ...error) Exception

	// GetSuppressed returns the list of suppressed errors that were intentionally
	// ignored or deferred while handling this exception. This can be useful when
	// multiple errors occur, but only one is chosen as the primary failure.
	GetSuppressed() []error

	// AddSuppressed attaches one or more suppressed errors to this exception.
	//
	// Note: This method may modify the current exception or return a new one. Always
	// use the returned [Exception].
	AddSuppressed(errors ...error) Exception

	// GetRecovered returns the value captured from a panic recovery, if any. It
	// returns nil if no value was recovered.
	GetRecovered() any

	// SetRecovered stores a recovered panic value inside this exception.
	//
	// Note: This method may modify the current exception or return a new one. Always
	// use the returned [Exception].
	SetRecovered(recovered any) Exception

	// GetStackTrace returns the stack trace captured for this exception, represented
	// as [StackFrames]. The result may be nil if no stack trace was filled.
	GetStackTrace() StackFrames

	// FillStackTrace captures the current call stack starting from the caller of
	// [FillStackTrace] itself and attaches it to this exception.
	//
	// The skip parameter controls how many additional stack frames are omitted. A
	// value of 0 includes the caller of [FillStackTrace], a value of 1 skips that
	// frame, and higher values skip more.
	//
	// Note: This method may modify the current exception or return a new one. Always
	// use the returned [Exception].
	FillStackTrace(skip int) Exception

	__() // private
}
