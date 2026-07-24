/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package common_test

import (
	"testing"

	"github.com/thanhminhmr/go-common/common"
)

// assertLeft asserts that o holds a left value equal to want: Left() reports
// it, Right() is absent, Either() reports state<0 with the same value, and
// Neither() is false.
func assertLeft[L, R comparable](t *testing.T, o common.Either[L, R], want L) {
	t.Helper()
	if value, exists := o.Left(); !exists {
		t.Errorf("Left() should exist")
	} else if value != want {
		t.Errorf("Left() = %v, want %v", value, want)
	}
	if _, exists := o.Right(); exists {
		t.Errorf("Right() should not exist")
	}
	if value, _, state := o.Either(); state >= 0 {
		t.Errorf("Either() state = %d, want < 0", state)
	} else if value != want {
		t.Errorf("Either() left = %v, want %v", value, want)
	}
	if o.Neither() {
		t.Errorf("Neither() should be false")
	}
}

// assertRight asserts that o holds a right value equal to want: Right() reports
// it, Left() is absent, Either() reports state>0 with the same value, and
// Neither() is false.
func assertRight[L, R comparable](t *testing.T, o common.Either[L, R], want R) {
	t.Helper()
	if _, exists := o.Left(); exists {
		t.Errorf("Left() should not exist")
	}
	if value, exists := o.Right(); !exists {
		t.Errorf("Right() should exist")
	} else if value != want {
		t.Errorf("Right() = %v, want %v", value, want)
	}
	if _, value, state := o.Either(); state <= 0 {
		t.Errorf("Either() state = %d, want > 0", state)
	} else if value != want {
		t.Errorf("Either() right = %v, want %v", value, want)
	}
	if o.Neither() {
		t.Errorf("Neither() should be false")
	}
}

type eitherStruct struct {
	a int
	b string
	c bool
}

func TestEitherNeither(t *testing.T) {
	o := common.Either[struct{}, struct{}]{}
	if _, exists := o.Left(); exists {
		t.Errorf("Left() should not exist")
	}
	if _, exists := o.Right(); exists {
		t.Errorf("Right() should not exist")
	}
	if _, _, state := o.Either(); state != 0 {
		t.Errorf("Either() should be neither")
	}
	if !o.Neither() {
		t.Errorf("Neither() should be true")
	}
}

func TestEitherLeft(t *testing.T) {
	t.Run("Nil", func(t *testing.T) { assertLeft(t, common.Left[any, any](nil), nil) })
	t.Run("Byte", func(t *testing.T) { assertLeft(t, common.Left[byte, byte](7), 7) })
	t.Run("DifferentTypes", func(t *testing.T) { assertLeft(t, common.Left[int, string](42), 42) })
	t.Run("Zero", func(t *testing.T) { assertLeft(t, common.Left[int, int](0), 0) })
	t.Run("EmptyString", func(t *testing.T) { assertLeft(t, common.Left[string, string](""), "") })
	t.Run("Struct", func(t *testing.T) {
		want := eitherStruct{a: 1, b: "hello", c: true}
		assertLeft(t, common.Left[eitherStruct, eitherStruct](want), want)
	})
}

func TestEitherRight(t *testing.T) {
	t.Run("Nil", func(t *testing.T) { assertRight(t, common.Right[any, any](nil), nil) })
	t.Run("Byte", func(t *testing.T) { assertRight(t, common.Right[byte, byte](7), 7) })
	t.Run("DifferentTypes", func(t *testing.T) { assertRight(t, common.Right[int, string]("hello"), "hello") })
	t.Run("Zero", func(t *testing.T) { assertRight(t, common.Right[int, int](0), 0) })
	t.Run("EmptyString", func(t *testing.T) { assertRight(t, common.Right[string, string](""), "") })
	t.Run("Struct", func(t *testing.T) {
		want := eitherStruct{a: 2, b: "world", c: false}
		assertRight(t, common.Right[eitherStruct, eitherStruct](want), want)
	})
}
