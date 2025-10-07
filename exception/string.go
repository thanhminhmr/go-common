package exception

import "fmt"

// type check
var _ Exception = String("")

// String is a string-based Exception. It behaves like a simple error containing
// only a message, with no causes, suppressed errors, recovered value, or stack
// trace.
//
// String is often used as a starting point for building a full exception with
// additional context. When causes, suppressed errors, or stack traces are added,
// a new Exception will be created that keeps the message and includes the added
// details:
//
//	err := exception.String("read failed").FillStackTrace(0)
//
// String can also be used as a constant error value, for example:
//
//	const ErrRead = exception.String("read failed")
type String string

func (e String) Error() string {
	return string(e)
}

func (e String) GetType() string {
	return string(e)
}

func (e String) GetMessage() string {
	return ""
}

func (e String) SetMessage(message string, parameters ...any) Exception {
	if message == "" {
		return e
	}
	if len(parameters) > 0 {
		return exception{
			Type:    string(e),
			Message: fmt.Sprintf(message, parameters...),
		}
	}
	return exception{
		Type:    string(e),
		Message: message,
	}
}

func (e String) GetCause() []error {
	return nil
}

func (e String) AddCause(errors ...error) Exception {
	var cause []error
	if combine(&cause, errors...) {
		return exception{
			Type:  string(e),
			Cause: cause,
		}
	}
	return e
}

func (e String) GetSuppressed() []error {
	return nil
}

func (e String) AddSuppressed(errors ...error) Exception {
	var suppressed []error
	if combine(&suppressed, errors...) {
		return exception{
			Type:       string(e),
			Suppressed: suppressed,
		}
	}
	return e
}

func (e String) GetRecovered() any {
	return nil
}

func (e String) SetRecovered(recovered any) Exception {
	if recovered == nil {
		return e
	}
	return exception{
		Type:      string(e),
		Recovered: recovered,
	}
}

func (e String) GetStackTrace() StackFrames {
	return nil
}

func (e String) FillStackTrace(skip int) Exception {
	return exception{
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
