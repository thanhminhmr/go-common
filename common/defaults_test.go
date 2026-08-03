/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package common_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thanhminhmr/go-common/common"
)

// =========================================================================
// Scalar defaults
// =========================================================================

func TestApplyDefaults_Scalars(t *testing.T) {
	type S struct {
		Str string     `default:"hello"`
		I   int        `default:"42"`
		I8  int8       `default:"-12"`
		U   uint       `default:"7"`
		F   float64    `default:"3.14"`
		B   bool       `default:"true"`
		C   complex128 `default:"1+2i"`
	}
	var s S
	require.NoError(t, common.ApplyDefaults(&s))
	assert.Equal(t, "hello", s.Str)
	assert.Equal(t, 42, s.I)
	assert.Equal(t, int8(-12), s.I8)
	assert.Equal(t, uint(7), s.U)
	assert.Equal(t, 3.14, s.F)
	assert.True(t, s.B)
	assert.Equal(t, 1+2i, s.C)
}

func TestApplyDefaults_EmptyStringDefault(t *testing.T) {
	type S struct {
		Name string `default:""`
	}
	var s S
	s.Name = "preexisting"
	require.NoError(t, common.ApplyDefaults(&s))
	assert.Equal(t, "", s.Name)
}

func TestApplyDefaults_OverwritesExistingValue(t *testing.T) {
	type S struct {
		Port int `default:"8080"`
	}
	s := S{Port: 9090}
	require.NoError(t, common.ApplyDefaults(&s))
	assert.Equal(t, 8080, s.Port)
}

// =========================================================================
// Pointer to scalar
// =========================================================================

func TestApplyDefaults_PointerToScalar(t *testing.T) {
	type S struct {
		Port *int `default:"8080"`
	}
	var s S
	require.NoError(t, common.ApplyDefaults(&s))
	require.NotNil(t, s.Port)
	assert.Equal(t, 8080, *s.Port)
}

// =========================================================================
// Nested structs
// =========================================================================

func TestApplyDefaults_NestedValueStruct(t *testing.T) {
	type Inner struct {
		Host string `default:"localhost"`
		Port int    `default:"5432"`
	}
	type Outer struct {
		Name   string `default:"app"`
		Inner  Inner
		SkipMe string
	}
	var o Outer
	require.NoError(t, common.ApplyDefaults(&o))
	assert.Equal(t, "app", o.Name)
	assert.Equal(t, "localhost", o.Inner.Host)
	assert.Equal(t, 5432, o.Inner.Port)
	assert.Empty(t, o.SkipMe)
}

func TestApplyDefaults_NestedPointerStruct_WithDefaults(t *testing.T) {
	type Inner struct {
		Host string `default:"localhost"`
	}
	type Outer struct {
		Inner *Inner
	}
	var o Outer
	require.NoError(t, common.ApplyDefaults(&o))
	require.NotNil(t, o.Inner)
	assert.Equal(t, "localhost", o.Inner.Host)
}

func TestApplyDefaults_NestedPointerStruct_NoDefaults_StaysNil(t *testing.T) {
	type Inner struct {
		Label string
	}
	type Outer struct {
		Inner *Inner
	}
	var o Outer
	require.NoError(t, common.ApplyDefaults(&o))
	assert.Nil(t, o.Inner,
		"ptr-to-struct with no defaults in subtree should not be allocated")
}

func TestApplyDefaults_DeeplyNested(t *testing.T) {
	type L3 struct {
		Val string `default:"deep"`
	}
	type L2 struct {
		L3 L3
	}
	type L1 struct {
		L2 L2
	}
	var l1 L1
	require.NoError(t, common.ApplyDefaults(&l1))
	assert.Equal(t, "deep", l1.L2.L3.Val)
}

// =========================================================================
// Embedded structs
// =========================================================================

func TestApplyDefaults_EmbeddedValueStruct(t *testing.T) {
	type Base struct {
		Host string `default:"localhost"`
	}
	type App struct {
		Base
		Port int `default:"8080"`
	}
	var a App
	require.NoError(t, common.ApplyDefaults(&a))
	assert.Equal(t, "localhost", a.Host)
	assert.Equal(t, 8080, a.Port)
}

func TestApplyDefaults_EmbeddedPointerStruct(t *testing.T) {
	type Base struct {
		Host string `default:"localhost"`
	}
	type App struct {
		*Base
		Port int `default:"8080"`
	}
	var a App
	require.NoError(t, common.ApplyDefaults(&a))
	require.NotNil(t, a.Base)
	assert.Equal(t, "localhost", a.Host)
	assert.Equal(t, 8080, a.Port)
}

// =========================================================================
// Custom types (TextUnmarshaler / mapstructure.Unmarshaler)
// =========================================================================

type defTextMode string

func (m *defTextMode) UnmarshalText(b []byte) error {
	*m = defTextMode(b)
	return nil
}

func TestApplyDefaults_TextUnmarshaler(t *testing.T) {
	type App struct {
		Mode defTextMode `default:"auto"`
	}
	var a App
	require.NoError(t, common.ApplyDefaults(&a))
	assert.Equal(t, defTextMode("auto"), a.Mode)
}

type defMSType struct {
	V int
}

func (m *defMSType) UnmarshalMapstructure(in any) error {
	if s, ok := in.(string); ok {
		m.V = len(s)
		return nil
	}
	return nil
}

func TestApplyDefaults_MapstructureUnmarshaler(t *testing.T) {
	type App struct {
		Val defMSType `default:"hello"`
	}
	var a App
	require.NoError(t, common.ApplyDefaults(&a))
	assert.Equal(t, 5, a.Val.V)
}

// =========================================================================
// Error cases
// =========================================================================

func TestApplyDefaults_InvalidDefault(t *testing.T) {
	type App struct {
		Port int `default:"abc"`
	}
	var a App
	err := common.ApplyDefaults(&a)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Port")
}

func TestApplyDefaults_DefaultOnStructField(t *testing.T) {
	type App struct {
		Sub struct{} `default:"x"`
	}
	var a App
	err := common.ApplyDefaults(&a)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Sub")
}

func TestApplyDefaults_ErrorIncludesFieldPath(t *testing.T) {
	type Inner struct {
		Port int `default:"abc"`
	}
	type Outer struct {
		Inner Inner
	}
	var o Outer
	err := common.ApplyDefaults(&o)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Inner.Port")
}

// =========================================================================
// Input validation
// =========================================================================

func TestApplyDefaults_NonPointerError(t *testing.T) {
	type S struct {
		X int `default:"1"`
	}
	require.Error(t, common.ApplyDefaults(S{}))
}

func TestApplyDefaults_NilPointerError(t *testing.T) {
	type S struct {
		X int `default:"1"`
	}
	var s *S
	require.Error(t, common.ApplyDefaults(s))
}

func TestApplyDefaults_DoublePointerRoot_Error(t *testing.T) {
	type S struct {
		X int `default:"42"`
	}
	var inner *S
	require.Error(t, common.ApplyDefaults(&inner))
}

// =========================================================================
// Edge cases
// =========================================================================

func TestApplyDefaults_NoDefaults_NoOp(t *testing.T) {
	type S struct {
		X int
		Y string
	}
	var s S
	require.NoError(t, common.ApplyDefaults(&s))
	assert.Equal(t, 0, s.X)
	assert.Empty(t, s.Y)
}

func TestApplyDefaults_UnexportedFieldSkipped(t *testing.T) {
	type S struct {
		Exported   string `default:"yes"`
		unexported string `default:"no"`
	}
	var s S
	require.NoError(t, common.ApplyDefaults(&s))
	assert.Equal(t, "yes", s.Exported)
	assert.Empty(t, s.unexported)
}

func TestApplyDefaults_RecursiveType(t *testing.T) {
	type Tree struct {
		Value int `default:"0"`
		Left  *Tree
		Right *Tree
	}
	var tr Tree
	err := common.ApplyDefaults(&tr)
	require.Error(t, err)
	assert.Equal(t, 0, tr.Value)
	// Left is processed first; its cyclic recursion hits maxDepth and errors,
	// so Right is never reached.
	require.NotNil(t, tr.Left)
	assert.Nil(t, tr.Right)
}

func TestApplyDefaults_CacheReuse(t *testing.T) {
	type S struct {
		X int `default:"42"`
	}
	var s1, s2 S
	require.NoError(t, common.ApplyDefaults(&s1))
	require.NoError(t, common.ApplyDefaults(&s2))
	assert.Equal(t, 42, s1.X)
	assert.Equal(t, 42, s2.X)
}

func TestApplyDefaults_PointerToNonStruct_Error(t *testing.T) {
	x := 42
	require.Error(t, common.ApplyDefaults(&x))
}

// =========================================================================
// All scalar sizes
// =========================================================================

func TestApplyDefaults_AllScalarSizes(t *testing.T) {
	type S struct {
		I8  int8    `default:"-1"`
		I16 int16   `default:"-2"`
		I32 int32   `default:"-3"`
		I64 int64   `default:"-4"`
		U8  uint8   `default:"1"`
		U16 uint16  `default:"2"`
		U32 uint32  `default:"3"`
		U64 uint64  `default:"4"`
		F32 float32 `default:"1.5"`
	}
	var s S
	require.NoError(t, common.ApplyDefaults(&s))
	assert.Equal(t, int8(-1), s.I8)
	assert.Equal(t, int16(-2), s.I16)
	assert.Equal(t, int32(-3), s.I32)
	assert.Equal(t, int64(-4), s.I64)
	assert.Equal(t, uint8(1), s.U8)
	assert.Equal(t, uint16(2), s.U16)
	assert.Equal(t, uint32(3), s.U32)
	assert.Equal(t, uint64(4), s.U64)
	assert.Equal(t, float32(1.5), s.F32)
}

// =========================================================================
// Multi-level pointer leaves
// =========================================================================

func TestApplyDefaults_DoublePointerLeaf(t *testing.T) {
	type S struct {
		Val **int `default:"42"`
	}
	var s S
	require.NoError(t, common.ApplyDefaults(&s))
	require.NotNil(t, s.Val)
	require.NotNil(t, *s.Val)
	assert.Equal(t, 42, **s.Val)
}

func TestApplyDefaults_TriplePointerLeaf(t *testing.T) {
	type S struct {
		Val ***int `default:"7"`
	}
	var s S
	require.NoError(t, common.ApplyDefaults(&s))
	require.NotNil(t, s.Val)
	require.NotNil(t, *s.Val)
	require.NotNil(t, **s.Val)
	assert.Equal(t, 7, ***s.Val)
}

// =========================================================================
// Mutual recursion (package-level types needed for cross-references)
// =========================================================================

type defMutA struct {
	Val int `default:"1"`
	B   *defMutB
}
type defMutB struct {
	Val int `default:"2"`
	A   *defMutA
}

func TestApplyDefaults_MutualRecursion(t *testing.T) {
	var a defMutA
	err := common.ApplyDefaults(&a)
	require.Error(t, err)
	assert.Equal(t, 1, a.Val)
	require.NotNil(t, a.B)
	assert.Equal(t, 2, a.B.Val)
	require.NotNil(t, a.B.A, "a.B.A allocated before depth cap reached")
}

// =========================================================================
// Embedded pointer with no defaults in subtree
// =========================================================================

func TestApplyDefaults_EmbeddedPointerNoDefaults(t *testing.T) {
	type Base struct {
		Label string
	}
	type App struct {
		*Base
		Port int `default:"8080"`
	}
	var a App
	require.NoError(t, common.ApplyDefaults(&a))
	assert.Nil(t, a.Base, "embedded ptr-to-struct with no defaults in subtree stays nil")
	assert.Equal(t, 8080, a.Port)
}

// =========================================================================
// Nil input
// =========================================================================

func TestApplyDefaults_NilInput(t *testing.T) {
	require.Error(t, common.ApplyDefaults(nil))
}

// =========================================================================
// Default on unsupported field types
// =========================================================================

func TestApplyDefaults_DefaultOnSlice(t *testing.T) {
	type S struct {
		Items []int `default:"x"`
	}
	var s S
	err := common.ApplyDefaults(&s)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Items")
}

func TestApplyDefaults_DefaultOnMap(t *testing.T) {
	type S struct {
		Data map[string]int `default:"x"`
	}
	var s S
	err := common.ApplyDefaults(&s)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Data")
}

// =========================================================================
// Pre-allocated pointers
// =========================================================================

func TestApplyDefaults_PreAllocatedPointerStruct(t *testing.T) {
	type Inner struct {
		Host string `default:"localhost"`
	}
	type Outer struct {
		Inner *Inner
	}
	inner := &Inner{Host: "override"}
	o := Outer{Inner: inner}
	require.NoError(t, common.ApplyDefaults(&o))
	assert.Same(t, inner, o.Inner, "should reuse pre-allocated pointer")
	assert.Equal(t, "localhost", o.Inner.Host)
}

func TestApplyDefaults_OverwritesPointerToScalar(t *testing.T) {
	type S struct {
		Port *int `default:"8080"`
	}
	v := 9090
	s := S{Port: &v}
	require.NoError(t, common.ApplyDefaults(&s))
	require.NotNil(t, s.Port)
	assert.Equal(t, 8080, *s.Port)
	assert.Equal(t, 8080, v, "should modify through the existing pointer")
}

// =========================================================================
// Multiple embeddings
// =========================================================================

func TestApplyDefaults_MultipleEmbeddings(t *testing.T) {
	type Base1 struct {
		Host string `default:"localhost"`
	}
	type Base2 struct {
		Port int `default:"8080"`
	}
	type App struct {
		Base1
		Base2
		Name string `default:"app"`
	}
	var a App
	require.NoError(t, common.ApplyDefaults(&a))
	assert.Equal(t, "localhost", a.Host)
	assert.Equal(t, 8080, a.Port)
	assert.Equal(t, "app", a.Name)
}

// =========================================================================
// Pointer to TextUnmarshaler
// =========================================================================

func TestApplyDefaults_PointerToTextUnmarshaler(t *testing.T) {
	type S struct {
		Mode *defTextMode `default:"auto"`
	}
	var s S
	require.NoError(t, common.ApplyDefaults(&s))
	require.NotNil(t, s.Mode)
	assert.Equal(t, defTextMode("auto"), *s.Mode)
}
