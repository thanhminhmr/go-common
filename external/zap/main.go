package zap

import "go.uber.org/zap/zapcore"

type Logger struct {
}

func (*Logger) Log(_ zapcore.Level, _ string, _ ...Field) {}

type Field interface{}
type Error interface{}

func String(_ ...any) Field {
	return nil
}
func Strings(_ ...any) Field {
	return nil
}
func Bool(_ ...any) Field {
	return nil
}
func Skip() Field {
	return nil
}
