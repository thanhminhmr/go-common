/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package common_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thanhminhmr/go-common/common"
)

// =========================================================================
// BindSimpleFromString
// =========================================================================

// msUnmarshalerOnly implements mapstructure.Unmarshaler (pointer receiver).
type msUnmarshalerOnly struct{ payload string }

func (m *msUnmarshalerOnly) UnmarshalMapstructure(in any) error {
	if s, ok := in.(string); ok {
		m.payload = s
		return nil
	}
	return fmt.Errorf("msUnmarshalerOnly: want string, got %T", in)
}

// msAlwaysErrUnmarshaler implements mapstructure.Unmarshaler and always fails.
type msAlwaysErrUnmarshaler struct{}

func (m *msAlwaysErrUnmarshaler) UnmarshalMapstructure(any) error {
	return fmt.Errorf("msAlwaysErrUnmarshaler: boom")
}

// msTextOnly implements encoding.TextUnmarshaler (pointer receiver).
type msTextOnly struct{ payload string }

func (t *msTextOnly) UnmarshalText(b []byte) error {
	t.payload = string(b)
	return nil
}

// msAlwaysErrText implements encoding.TextUnmarshaler and always fails.
type msAlwaysErrText struct{}

func (t *msAlwaysErrText) UnmarshalText([]byte) error { return fmt.Errorf("msAlwaysErrText: boom") }

// msUnmarshalAndText implements both interfaces; UnmarshalMapstructure must win.
type msUnmarshalAndText struct {
	muCalled bool
	tuCalled bool
	payload  string
}

func (u *msUnmarshalAndText) UnmarshalMapstructure(in any) error {
	u.muCalled = true
	if s, ok := in.(string); ok {
		u.payload = s
	}
	return nil
}

func (u *msUnmarshalAndText) UnmarshalText(b []byte) error {
	u.tuCalled = true
	u.payload = string(b)
	return nil
}

func TestBindSimpleFromString(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		var v string
		require.NoError(t, common.BindSimpleFromString("hello", &v))
		assert.Equal(t, "hello", v)
	})

	t.Run("ints", func(t *testing.T) {
		cases := []struct {
			name string
			typ  reflect.Type
			in   string
			want any
		}{
			{"int", reflect.TypeFor[int](), "42", 42},
			{"int8", reflect.TypeFor[int8](), "-12", int8(-12)},
			{"int16", reflect.TypeFor[int16](), "1000", int16(1000)},
			{"int32", reflect.TypeFor[int32](), "-100000", int32(-100000)},
			{"int64", reflect.TypeFor[int64](), "9000000000", int64(9000000000)},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				target := reflect.New(c.typ)
				require.NoError(t, common.BindSimpleFromString(c.in, target.Interface()))
				assert.Equal(t, c.want, target.Elem().Interface())
			})
		}
	})

	t.Run("int overflow error", func(t *testing.T) {
		var v int8
		require.Error(t, common.BindSimpleFromString("200", &v))
	})

	t.Run("int invalid error", func(t *testing.T) {
		var v int
		require.Error(t, common.BindSimpleFromString("abc", &v))
	})

	t.Run("uints", func(t *testing.T) {
		cases := []struct {
			name string
			typ  reflect.Type
			in   string
			want any
		}{
			{"uint", reflect.TypeFor[uint](), "42", uint(42)},
			{"uint8", reflect.TypeFor[uint8](), "255", uint8(255)},
			{"uint16", reflect.TypeFor[uint16](), "60000", uint16(60000)},
			{"uint32", reflect.TypeFor[uint32](), "4000000000", uint32(4000000000)},
			{"uint64", reflect.TypeFor[uint64](), "9000000000", uint64(9000000000)},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				target := reflect.New(c.typ)
				require.NoError(t, common.BindSimpleFromString(c.in, target.Interface()))
				assert.Equal(t, c.want, target.Elem().Interface())
			})
		}
	})

	t.Run("uint negative error", func(t *testing.T) {
		var v uint
		require.Error(t, common.BindSimpleFromString("-1", &v))
	})

	t.Run("floats", func(t *testing.T) {
		cases := []struct {
			name string
			typ  reflect.Type
			in   string
			want any
		}{
			{"float32", reflect.TypeFor[float32](), "3.14", float32(3.14)},
			{"float64", reflect.TypeFor[float64](), "2.71828", 2.71828},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				target := reflect.New(c.typ)
				require.NoError(t, common.BindSimpleFromString(c.in, target.Interface()))
				assert.Equal(t, c.want, target.Elem().Interface())
			})
		}
	})

	t.Run("float invalid error", func(t *testing.T) {
		var v float64
		require.Error(t, common.BindSimpleFromString("abc", &v))
	})

	t.Run("bool", func(t *testing.T) {
		var v bool
		require.NoError(t, common.BindSimpleFromString("true", &v))
		assert.True(t, v)

		v = false
		require.NoError(t, common.BindSimpleFromString("true", &v))
		assert.True(t, v)

		v = true
		require.NoError(t, common.BindSimpleFromString("false", &v))
		assert.False(t, v)
	})

	t.Run("bool invalid error", func(t *testing.T) {
		var v bool
		require.Error(t, common.BindSimpleFromString("maybe", &v))
	})

	t.Run("complex", func(t *testing.T) {
		cases := []struct {
			name string
			typ  reflect.Type
			want any
		}{
			{"complex64", reflect.TypeFor[complex64](), complex64(1 + 2i)},
			{"complex128", reflect.TypeFor[complex128](), 1 + 2i},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				target := reflect.New(c.typ)
				require.NoError(t, common.BindSimpleFromString("1+2i", target.Interface()))
				assert.Equal(t, c.want, target.Elem().Interface())
			})
		}
	})

	t.Run("complex invalid error", func(t *testing.T) {
		var v complex128
		require.Error(t, common.BindSimpleFromString("abc", &v))
	})

	t.Run("mapstructure.Unmarshaler (pointer receiver)", func(t *testing.T) {
		var out msUnmarshalerOnly
		require.NoError(t, common.BindSimpleFromString("hello", &out))
		assert.Equal(t, "hello", out.payload)
	})

	t.Run("mapstructure.Unmarshaler error propagates", func(t *testing.T) {
		var out msAlwaysErrUnmarshaler
		err := common.BindSimpleFromString("hello", &out)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "boom")
	})

	t.Run("encoding.TextUnmarshaler (pointer receiver)", func(t *testing.T) {
		var out msTextOnly
		require.NoError(t, common.BindSimpleFromString("hello", &out))
		assert.Equal(t, "hello", out.payload)
	})

	t.Run("encoding.TextUnmarshaler error propagates", func(t *testing.T) {
		var out msAlwaysErrText
		err := common.BindSimpleFromString("hello", &out)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "boom")
	})

	t.Run("Unmarshaler wins over TextUnmarshaler", func(t *testing.T) {
		var out msUnmarshalAndText
		require.NoError(t, common.BindSimpleFromString("hello", &out))
		assert.True(t, out.muCalled)
		assert.False(t, out.tuCalled)
		assert.Equal(t, "hello", out.payload)
	})

	t.Run("unknown type returns error", func(t *testing.T) {
		var out struct{}
		err := common.BindSimpleFromString("hello", &out)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown target type")
	})
}

// =========================================================================
// BindStructWithTag
// =========================================================================

type msTagSelect struct {
	Foo int `foo:"x"`
	Bar int `bar:"x"`
}

type msUntagged struct {
	Tagged   int `t:"tagged"`
	Untagged int
}

type msCaseSensitive struct {
	Foo int `t:"foo"`
}

type msEmbedded struct {
	Inner int `t:"inner"`
}

type msOuter struct {
	msEmbedded
	Outer int `t:"outer"`
}

type msSquashInner struct {
	A int `t:"a"`
}

// msSquashTagged has a named field tagged ",squash"; the sentinel in
// BindStructWithTag disables that option, so the field stays nested under its
// field name "Inner".
type msSquashTagged struct {
	Inner msSquashInner `t:",squash"`
}

type msWrapSlice struct {
	V []int `t:"v"`
}

type msUnwrapSlice struct {
	V int `t:"v"`
}

type msMySlice []int

type msBothSlice struct {
	V msMySlice `t:"v"`
}

type msMyUnmarshaler struct {
	V int
}

func (m *msMyUnmarshaler) UnmarshalMapstructure(in any) error {
	if s, ok := in.(string); ok {
		m.V = len(s)
		return nil
	}
	return fmt.Errorf("msMyUnmarshaler: want string, got %T", in)
}

type msUnmarshalerField struct {
	F msMyUnmarshaler `t:"f"`
}

type msTextField struct {
	F msTextOnly `t:"f"`
}

type msStringToInt struct {
	V int `t:"v"`
}

type msNumberToInt struct {
	V int `t:"v"`
}

type msSubStruct struct {
	V int `t:"v"`
}

type msNestedStruct struct {
	Sub msSubStruct `t:"sub"`
}

type msUnknownField struct {
	V struct{} `t:"v"`
}

func TestBindStructWithTag(t *testing.T) {
	t.Run("TagName selects field by tag", func(t *testing.T) {
		var out msTagSelect
		require.NoError(t, common.BindStructWithTag("foo", map[string]any{"x": 1}, &out))
		assert.Equal(t, 1, out.Foo)
		assert.Equal(t, 0, out.Bar)

		out = msTagSelect{}
		require.NoError(t, common.BindStructWithTag("bar", map[string]any{"x": 1}, &out))
		assert.Equal(t, 0, out.Foo)
		assert.Equal(t, 1, out.Bar)
	})

	t.Run("IgnoreUntaggedFields leaves untagged field zero", func(t *testing.T) {
		var out msUntagged
		require.NoError(t, common.BindStructWithTag("t", map[string]any{
			"tagged":   1,
			"Untagged": 2,
		}, &out))
		assert.Equal(t, 1, out.Tagged)
		assert.Equal(t, 0, out.Untagged)
	})

	t.Run("MatchName is case-sensitive", func(t *testing.T) {
		var lower msCaseSensitive
		require.NoError(t, common.BindStructWithTag("t", map[string]any{"foo": 1}, &lower))
		assert.Equal(t, 1, lower.Foo)

		var upper msCaseSensitive
		require.NoError(t, common.BindStructWithTag("t", map[string]any{"Foo": 1}, &upper))
		assert.Equal(t, 0, upper.Foo)
	})

	t.Run("Squash promotes embedded anonymous fields", func(t *testing.T) {
		var out msOuter
		require.NoError(t, common.BindStructWithTag("t", map[string]any{
			"inner": 1,
			"outer": 2,
		}, &out))
		assert.Equal(t, 1, out.Inner)
		assert.Equal(t, 2, out.Outer)
	})

	t.Run("SquashTagOption sentinel disables tag-squash", func(t *testing.T) {
		var nested msSquashTagged
		require.NoError(t, common.BindStructWithTag("t", map[string]any{
			"Inner": map[string]any{"a": 1},
		}, &nested))
		assert.Equal(t, 1, nested.Inner.A)

		var promoted msSquashTagged
		require.NoError(t, common.BindStructWithTag("t", map[string]any{"a": 1}, &promoted))
		assert.Equal(t, 0, promoted.Inner.A)
	})

	t.Run("same type scalar short-circuits", func(t *testing.T) {
		var out msStringToInt
		require.NoError(t, common.BindStructWithTag("t", map[string]any{"v": 42}, &out))
		assert.Equal(t, 42, out.V)
	})

	t.Run("scalar wrapped into single-element slice", func(t *testing.T) {
		var out msWrapSlice
		require.NoError(t, common.BindStructWithTag("t", map[string]any{"v": 42}, &out))
		assert.Equal(t, []int{42}, out.V)
	})

	t.Run("single-element slice unwrapped to scalar", func(t *testing.T) {
		var out msUnwrapSlice
		require.NoError(t, common.BindStructWithTag("t", map[string]any{"v": []int{42}}, &out))
		assert.Equal(t, 42, out.V)
	})

	t.Run("both pure slices pass through", func(t *testing.T) {
		var out msBothSlice
		require.NoError(t, common.BindStructWithTag("t", map[string]any{"v": []int{1, 2, 3}}, &out))
		assert.Equal(t, msMySlice{1, 2, 3}, out.V)
	})

	t.Run("target implements mapstructure.Unmarshaler", func(t *testing.T) {
		var out msUnmarshalerField
		require.NoError(t, common.BindStructWithTag("t", map[string]any{"f": "hello"}, &out))
		assert.Equal(t, 5, out.F.V)
	})

	t.Run("target implements encoding.TextUnmarshaler", func(t *testing.T) {
		var out msTextField
		require.NoError(t, common.BindStructWithTag("t", map[string]any{"f": "hello"}, &out))
		assert.Equal(t, "hello", out.F.payload)
	})

	t.Run("string to int via BindSimpleFromString", func(t *testing.T) {
		var out msStringToInt
		require.NoError(t, common.BindStructWithTag("t", map[string]any{"v": "42"}, &out))
		assert.Equal(t, 42, out.V)
	})

	t.Run("json.Number to int via BindSimpleFromString", func(t *testing.T) {
		var out msNumberToInt
		require.NoError(t, common.BindStructWithTag("t", map[string]any{"v": json.Number("42")}, &out))
		assert.Equal(t, 42, out.V)
	})

	t.Run("fallthrough decodes nested map to struct", func(t *testing.T) {
		var out msNestedStruct
		require.NoError(t, common.BindStructWithTag("t", map[string]any{
			"sub": map[string]any{"v": 1},
		}, &out))
		assert.Equal(t, 1, out.Sub.V)
	})

	t.Run("unknown target then fallthrough yields error", func(t *testing.T) {
		var out msUnknownField
		err := common.BindStructWithTag("t", map[string]any{"v": "hello"}, &out)
		require.Error(t, err)
	})

	t.Run("parse error propagates", func(t *testing.T) {
		var out msStringToInt
		err := common.BindStructWithTag("t", map[string]any{"v": "abc"}, &out)
		require.Error(t, err)
	})

	t.Run("Unmarshaler error propagates", func(t *testing.T) {
		var out msUnmarshalerField
		err := common.BindStructWithTag("t", map[string]any{"f": 123}, &out)
		require.Error(t, err)
	})
}
