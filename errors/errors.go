package errors

import (
	"fmt"
	"slices"

	"github.com/rs/zerolog"
)

type Error interface {
	error

	GetCause() []error
	AddCause(...error) Error

	GetSuppressed() []error
	AddSuppressed(...error) Error

	GetRecovered() any
	SetRecovered(any) Error

	GetStackTrace() []StackFrame
	FillStackTrace(skip int) Error
}

// ========================================

func Errors(errors ...error) Error {
	err := String("multiple errors").AddCause(errors...)
	if len(err.GetCause()) > 0 {
		return err
	}
	return nil
}

// ========================================

type Template string

func (e Template) Format(arg ...any) String {
	return String(fmt.Sprintf(string(e), arg...))
}

// ========================================

type String string

func (e String) MarshalZerologObject(event *zerolog.Event) {
	event.Str("error", string(e))
}

func (e String) Error() string {
	return string(e)
}

func (e String) GetCause() []error {
	return nil
}

func (e String) AddCause(errors ...error) Error {
	if len(errors) == 0 {
		return e
	}
	cause := slices.DeleteFunc(errors, func(err error) bool { return err == nil })
	if len(cause) == 0 {
		return e
	}
	return fullError{
		String: string(e),
		Cause:  cause,
	}
}

func (e String) GetSuppressed() []error {
	return nil
}

func (e String) AddSuppressed(errors ...error) Error {
	if len(errors) == 0 {
		return e
	}
	suppressed := slices.DeleteFunc(errors, func(err error) bool { return err == nil })
	if len(suppressed) == 0 {
		return e
	}
	return fullError{
		String:     string(e),
		Suppressed: suppressed,
	}
}

func (e String) GetRecovered() any {
	return nil
}

func (e String) SetRecovered(recovered any) Error {
	if recovered == nil {
		return e
	}
	return fullError{
		String:    string(e),
		Recovered: recovered,
	}
}

func (e String) GetStackTrace() []StackFrame {
	return nil
}

func (e String) FillStackTrace(skip int) Error {
	return fullError{
		String:     string(e),
		StackTrace: StackFrames(skip + 1),
	}
}

// ========================================

type fullError struct {
	String     string
	Cause      []error
	Suppressed []error
	Recovered  any
	StackTrace []StackFrame
}

func (e fullError) MarshalZerologObject(event *zerolog.Event) {
	event = event.Str("error", e.String)
	if e.Cause != nil {
		event = event.Errs("cause", e.Cause)
	}
	if e.Suppressed != nil {
		event = event.Errs("suppressed", e.Suppressed)
	}
	if e.Recovered != nil {
		event = event.Any("recovered", e.Recovered)
	}
	if e.StackTrace != nil {
		event = event.Any("stack_trace", e.StackTrace)
	}
}

func (e fullError) Error() string {
	return e.String
}

func (e fullError) GetCause() []error {
	return e.Cause
}

func (e fullError) AddCause(errors ...error) Error {
	if len(errors) == 0 {
		return e
	}
	if len(e.Cause) == 0 {
		e.Cause = slices.DeleteFunc(errors, func(err error) bool { return err == nil })
		return e
	}
	e.Cause = slices.Grow(e.Cause, len(errors))
	for _, inner := range errors {
		if inner != nil {
			e.Cause = append(e.Cause, inner)
		}
	}
	return e
}

func (e fullError) GetSuppressed() []error {
	return e.Suppressed
}

func (e fullError) AddSuppressed(errors ...error) Error {
	if len(errors) == 0 {
		return e
	}
	if len(e.Suppressed) == 0 {
		e.Suppressed = slices.DeleteFunc(errors, func(err error) bool { return err == nil })
		return e
	}
	e.Suppressed = slices.Grow(e.Suppressed, len(errors))
	for _, inner := range errors {
		if inner != nil {
			e.Suppressed = append(e.Suppressed, inner)
		}
	}
	return e
}

func (e fullError) GetRecovered() any {
	return e.Recovered
}

func (e fullError) SetRecovered(recovered any) Error {
	e.Recovered = recovered
	return e
}

func (e fullError) GetStackTrace() []StackFrame {
	return e.StackTrace
}

func (e fullError) FillStackTrace(skip int) Error {
	e.StackTrace = StackFrames(skip + 1)
	return e
}
