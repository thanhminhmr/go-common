package errors

import "errors"

const ErrUnsupported = String("unsupported operation")

func New(message string) error {
	return String(message)
}

func Join(errors ...error) error {
	return Errors(errors...)
}

func Unwrap(err error) error {
	return errors.Unwrap(err)
}

func Is(err error, target error) bool {
	return errors.Is(err, target)
}

func As(err error, target any) bool {
	return errors.As(err, target)
}
