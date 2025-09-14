package errors

type Error interface {
	error

	GetCause() []error
	AddCause(errors ...error) Error

	GetSuppressed() []error
	AddSuppressed(errors ...error) Error

	GetRecovered() any
	SetRecovered(recovered any) Error

	GetStackTrace() []StackFrame
	FillStackTrace(skip int) Error

	Is(target error) bool
	As(target any) bool
}
