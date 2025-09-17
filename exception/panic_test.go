package exception_test

import (
	"testing"

	"github.com/thanhminhmr/go-common/exception"
)

func TestPanicRecoverPair(t *testing.T) {
	defer func() {
		if recovered := exception.Recover(recover()); recovered != nil {
			checkStackTrace(t, recovered.GetStackTrace(), "/exception_test.TestPanicRecoverPair")
		}
	}()
	exception.Panic("Test")
}

func TestRecoverRawPanic(t *testing.T) {
	defer func() {
		if recovered := exception.Recover(recover()); recovered != nil {
			checkStackTrace(t, recovered.GetStackTrace(), "/exception_test.TestRecoverRawPanic")
		}
	}()
	panic("Test")
}
