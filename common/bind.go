/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package common

import (
	"encoding"
	"encoding/json"
	"reflect"
	"strconv"
	"unsafe"

	"github.com/go-viper/mapstructure/v2"
)

// BindStructWithTag decodes input into the struct pointed to by output, using
// the named struct tag to match map keys to exported fields. Matching is
// case-sensitive, and fields without the tag are ignored. Anonymous embedded
// structs are squashed (their fields promoted to the parent); the `,squash`
// tag option is not honored.
//
// In addition to mapstructure's built-in conversions, target types implementing
// [mapstructure.Unmarshaler] or [encoding.TextUnmarshaler] are honored, and a
// string or [json.Number] source is accepted for built-in scalar targets. A
// scalar source may be wrapped into a single-element slice, and a
// single-element slice source may be unwrapped onto a scalar.
func BindStructWithTag(tag string, input, output any) error {
	if decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: internalDecodeHookFunc,
		Squash:     true,
		Result:     output,
		TagName:    tag,
		// sentinel that should not match any real tag name, disabling squash-by-tag
		SquashTagOption: "\xFF\xFF\xFF\xFF",
		// making name matching case-sensitive
		MatchName:            func(a, b string) bool { return a == b },
		IgnoreUntaggedFields: true,
	}); err != nil {
		return err
	} else {
		return decoder.Decode(input)
	}
}

// BindSimpleFromString parses input and stores the result in the value pointed
// to by output. output must be a pointer (or interface) whose element is one of:
//
//   - a built-in scalar ([string], [bool], any [int], [uint], [float], or
//     [complex] width);
//   - a type implementing [mapstructure.Unmarshaler]; or
//   - a type implementing [encoding.TextUnmarshaler].
//
// For any other target type BindSimpleFromString returns an error.
func BindSimpleFromString(input string, output any) error {
	switch target := reflect.ValueOf(output); target.Kind() {
	case reflect.Pointer, reflect.Interface:
		return internalDecodeString(input, target.Elem())
	default:
		return unknownOutputType{}
	}
}

func internalDecodeString(input string, output reflect.Value) error {
	switch unmarshaller := output.Addr().Interface().(type) {
	case mapstructure.Unmarshaler:
		return unmarshaller.UnmarshalMapstructure(input)
	case encoding.TextUnmarshaler:
		return unmarshaller.UnmarshalText(unsafeStringToBytes(input))
	case *string:
		output.SetString(input)
	case *bool:
		if parsed, err := strconv.ParseBool(input); err != nil {
			return err
		} else {
			output.SetBool(parsed)
		}
	case *int, *int8, *int16, *int32, *int64:
		if parsed, err := strconv.ParseInt(input, 0, output.Type().Bits()); err != nil {
			return err
		} else {
			output.SetInt(parsed)
		}
	case *uint, *uint8, *uint16, *uint32, *uint64:
		if parsed, err := strconv.ParseUint(input, 0, output.Type().Bits()); err != nil {
			return err
		} else {
			output.SetUint(parsed)
		}
	case *float32, *float64:
		if parsed, err := strconv.ParseFloat(input, output.Type().Bits()); err != nil {
			return err
		} else {
			output.SetFloat(parsed)
		}
	case *complex64, *complex128:
		if parsed, err := strconv.ParseComplex(input, output.Type().Bits()); err != nil {
			return err
		} else {
			output.SetComplex(parsed)
		}
	default:
		return unknownOutputType{}
	}
	return nil
}

func internalDecodeHookFunc(source, target reflect.Value) (any, error) {
	// same type, no need to change
	if source.Type() == target.Type() {
		target.Set(source)
		return nil, nil
	}
	{ // check if source and target is pure slice
		sourceIsPureSlice := source.Kind() == reflect.Slice && source.NumMethod() == 0
		targetIsPureSlice := target.Kind() == reflect.Slice && target.NumMethod() == 0 && target.Addr().NumMethod() == 0
		// if both are pure slice, short circuit
		if sourceIsPureSlice && targetIsPureSlice {
			return source.Interface(), nil
		}
		// if source is not a pure slice and target is a pure slice
		if targetIsPureSlice {
			// wrap source into a single element slice
			wrapper := reflect.MakeSlice(reflect.SliceOf(source.Type()), 1, 1)
			wrapper.Index(0).Set(source)
			return wrapper.Interface(), nil
		}
		// if source is a pure slice with single element, and target is not a pure slice
		if sourceIsPureSlice && source.Len() == 1 {
			source = source.Index(0)
		}
	}
	// check if target implements mapstructure.Unmarshaler
	if targetUnmarshaller, ok := reflect.TypeAssert[mapstructure.Unmarshaler](target.Addr()); ok {
		if err := targetUnmarshaller.UnmarshalMapstructure(source.Interface()); err != nil {
			return nil, err
		}
		return nil, nil
	}
	// check if input is string or json.Number
	switch source.Interface().(type) {
	case string, json.Number:
		if err := internalDecodeString(source.String(), target); err != nil {
			//goland:noinspection GoTypeAssertionOnErrors
			if _, ok := err.(unknownOutputType); ok {
				break
			}
			return nil, err
		}
		return nil, nil
	}
	return source.Interface(), nil
}

type unknownOutputType struct{}

func (u unknownOutputType) Error() string { return "unknown target type" }

func unsafeStringToBytes(value string) []byte {
	return unsafe.Slice(unsafe.StringData(value), len(value))
}
