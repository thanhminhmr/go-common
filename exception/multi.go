package exception

// Join combines multiple errors into a single Exception.
//
// Nil values are ignored. If no errors remain, Join returns nil. If there is
// exactly one non-nil error, and it already implements Exception, it is returned
// directly.
//
// Otherwise, Join creates a new Exception that represents multiple causes. This
// special Exception only holds the underlying errors; other details such as
// suppressed errors, recovered value, and stack trace are left empty.
//
// Note: The returned Exception may reuse or wrap the given errors. Callers
// should always use the returned value rather than assuming the original inputs
// remain unchanged.
func Join(errors ...error) Exception {
	var multiple []error
	if !combine(&multiple, errors...) {
		return nil
	}
	if len(multiple) == 1 {
		if inner, ok := multiple[0].(Exception); ok {
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

func (e multipleErrors) AddCause(errors ...error) Exception {
	concat((*[]error)(&e), errors...)
	return e
}

func (e multipleErrors) GetSuppressed() []error {
	return nil
}

func (e multipleErrors) AddSuppressed(errors ...error) Exception {
	var suppressed []error
	if combine(&suppressed, errors...) {
		return exception{
			Cause:      e,
			Suppressed: suppressed,
		}
	}
	return e
}

func (e multipleErrors) GetRecovered() any {
	return nil
}

func (e multipleErrors) SetRecovered(recovered any) Exception {
	if recovered == nil {
		return e
	}
	return exception{
		Cause:     e,
		Recovered: recovered,
	}
}

func (e multipleErrors) GetStackTrace() StackFrames {
	return nil
}

func (e multipleErrors) FillStackTrace(skip int) Exception {
	return exception{
		Cause:      e,
		StackTrace: StackTrace(skip + 1),
	}
}

func (e multipleErrors) __() {}

func (e multipleErrors) Unwrap() []error {
	return e
}
