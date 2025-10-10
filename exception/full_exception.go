package exception

import "fmt"

// type check
var _ Exception = fullException{}

type fullException struct {
	Type       string
	Message    string
	Cause      []error
	Suppressed []error
	Recovered  any
	StackTrace []StackFrame
}

func (e fullException) Error() string {
	switch {
	case e.Type == "":
		return e.Message
	case e.Message == "":
		return e.Type
	default:
		return e.Type + ": " + e.Message
	}
}

func (e fullException) GetType() string {
	return e.Type
}

func (e fullException) GetMessage() string {
	return e.Message
}

func (e fullException) SetMessage(message string, parameters ...any) Exception {
	if len(parameters) > 0 {
		e.Message = fmt.Sprintf(message, parameters...)
	} else {
		e.Message = message
	}
	return e
}

func (e fullException) GetCause() []error {
	return e.Cause
}

func (e fullException) AddCause(errors ...error) Exception {
	concat(&e.Cause, errors...)
	return e
}

func (e fullException) GetSuppressed() []error {
	return e.Suppressed
}

func (e fullException) AddSuppressed(errors ...error) Exception {
	concat(&e.Suppressed, errors...)
	return e
}

func (e fullException) GetRecovered() any {
	return e.Recovered
}

func (e fullException) SetRecovered(recovered any) Exception {
	e.Recovered = recovered
	return e
}

func (e fullException) GetStackTrace() StackFrames {
	return e.StackTrace
}

func (e fullException) FillStackTrace(skip int) Exception {
	e.StackTrace = StackTrace(skip + 1)
	return e
}

func (e fullException) __() {}

func (e fullException) Unwrap() []error {
	return e.Cause
}

func (e fullException) Is(target error) bool {
	return is(e, target)
}

func (e fullException) As(target any) bool {
	return as(e, target)
}
