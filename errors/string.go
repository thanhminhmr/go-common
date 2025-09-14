package errors

type String string

func (e String) Error() string {
	return string(e)
}

func (e String) GetCause() []error {
	return nil
}

func (e String) AddCause(errors ...error) Error {
	var cause []error
	if combine(&cause, errors...) {
		return fullError{
			String: string(e),
			Cause:  cause,
		}
	}
	return e
}

func (e String) GetSuppressed() []error {
	return nil
}

func (e String) AddSuppressed(errors ...error) Error {
	var suppressed []error
	if combine(&suppressed, errors...) {
		return fullError{
			String:     string(e),
			Suppressed: suppressed,
		}
	}
	return e
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
		StackTrace: StackTrace(skip + 1),
	}
}

func (e String) Is(target error) bool {
	return is(e, target)
}

func (e String) As(target any) bool {
	return as(e, target)
}
