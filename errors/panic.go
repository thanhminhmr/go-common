package errors

const PanicError = String("panicked")

func Panic(recovered any, skip int) {
	if err, ok := recovered.(Error); ok {
		if err.GetStackTrace() == nil {
			recovered = err.FillStackTrace(skip + 1)
		}
	} else {
		recovered = fullError{
			String:     string(PanicError),
			Recovered:  recovered,
			StackTrace: StackTrace(skip + 1),
		}
	}
	panic(recovered)
}

func Recover(skip int) Error {
	if recovered := recover(); recovered != nil {
		if err, ok := recovered.(Error); ok {
			if err.GetStackTrace() == nil {
				err = err.FillStackTrace(skip + 1)
			}
			return err
		}
		return fullError{
			String:     string(PanicError),
			Recovered:  recovered,
			StackTrace: StackTrace(skip + 1),
		}
	}
	return nil
}
