package exception_test

import (
	"strings"
	"testing"

	"github.com/thanhminhmr/go-common/exception"
)

func TestStackTrace(t *testing.T) {
	trace := exception.StackTrace(0)
	if len(trace) == 0 {
		t.Fatalf("expected non-empty stack trace")
	}
	for _, frame := range trace {
		if frame.Function == "" || frame.File == "" || frame.Line == 0 {
			t.Fatalf("expected function, file, and line populated, got %#v", frame)
		}
	}
	if !strings.HasSuffix(trace[0].Function, "/exception_test.TestStackTrace") {
		t.Fatalf("expected first function is this function, got %#v", trace[0])
	}
}
