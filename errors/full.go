package errors

type fullError struct {
	String     string
	Cause      []error
	Suppressed []error
	Recovered  any
	StackTrace []StackFrame
}

func (e fullError) Error() string {
	return e.String
}

func (e fullError) GetCause() []error {
	return e.Cause
}

func (e fullError) AddCause(errors ...error) Error {
	concat(&e.Cause, errors...)
	return e
}

func (e fullError) GetSuppressed() []error {
	return e.Suppressed
}

func (e fullError) AddSuppressed(errors ...error) Error {
	concat(&e.Suppressed, errors...)
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
	e.StackTrace = StackTrace(skip + 1)
	return e
}

func (e fullError) Unwrap() []error {
	return e.Cause
}

func (e fullError) Is(target error) bool {
	return is(e, target)
}

func (e fullError) As(target any) bool {
	return as(e, target)
}
