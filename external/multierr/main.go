package multierr

import "github.com/thanhminhmr/go-common/errors"

func Combine(values ...error) error {
	return errors.Errors(values...)
}

func Append(first error, second error) error {
	return errors.Errors(first, second)
}
