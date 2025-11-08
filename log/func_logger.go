package log

import (
	"fmt"
	"reflect"
	"runtime"

	"github.com/rs/zerolog"
)

func FuncOrAny(e *zerolog.Event, k string, v any) *zerolog.Event {
	r := reflect.ValueOf(v)
	for r.Kind() == reflect.Ptr || r.Kind() == reflect.Interface {
		r = r.Elem()
	}
	if r.Kind() == reflect.Func {
		return e.Stringer(k, _func{v: r.Interface()})
	}
	return e.Any(k, v)
}

func Func(v any) fmt.Stringer {
	return _func{v: v}
}

func Funcs[S ~[]E, E any](v S) zerolog.LogArrayMarshaler {
	return _funcs[S, E]{v: v}
}

type _func struct {
	v any
	s string
}

func (f _func) String() string {
	if f.v == nil {
		return "<nil>"
	}
	v := reflect.ValueOf(f.v)
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	if v.Kind() != reflect.Func {
		return "<invalid>"
	}
	funcForPC := runtime.FuncForPC(v.Pointer())
	if funcForPC == nil {
		return "<unknown>"
	}
	file, line := funcForPC.FileLine(funcForPC.Entry())
	return fmt.Sprintf("%s() @ %s:%d", funcForPC.Name(), file, line)
}

type _funcs[S ~[]E, E any] struct {
	v S
}

func (f _funcs[S, E]) MarshalZerologArray(array *zerolog.Array) {
	if len(f.v) == 0 {
		return
	}
	for _, v := range f.v {
		array = array.Str(_func{v: v}.String())
	}
}
