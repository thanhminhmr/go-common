package errors

func Errors(errors ...error) Error {
	var multiple []error
	if !combine(&multiple, errors...) {
		return nil
	}
	if len(multiple) == 1 {
		if inner, ok := multiple[0].(Error); ok {
			return inner
		}
	}
	return multipleErrors(multiple)
}

type multipleErrors []error

func (e multipleErrors) Error() string {
	return ""
}

func (e multipleErrors) GetCause() []error {
	return e
}

func (e multipleErrors) AddCause(errors ...error) Error {
	concat((*[]error)(&e), errors...)
	return e
}

func (e multipleErrors) GetSuppressed() []error {
	return nil
}

func (e multipleErrors) AddSuppressed(errors ...error) Error {
	var suppressed []error
	if combine(&suppressed, errors...) {
		return fullError{
			Cause:      e,
			Suppressed: suppressed,
		}
	}
	return e
}

func (e multipleErrors) GetRecovered() any {
	return nil
}

func (e multipleErrors) SetRecovered(recovered any) Error {
	if recovered == nil {
		return e
	}
	return fullError{
		Cause:     e,
		Recovered: recovered,
	}
}

func (e multipleErrors) GetStackTrace() []StackFrame {
	return nil
}

func (e multipleErrors) FillStackTrace(skip int) Error {
	return fullError{
		Cause:      e,
		StackTrace: StackTrace(skip + 1),
	}
}

func (e multipleErrors) Unwrap() []error {
	return e
}

func (e multipleErrors) Is(error) bool {
	return false
}

func (e multipleErrors) As(any) bool {
	return false
}
