/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package either_test

import (
	"reflect"
	"testing"

	"github.com/thanhminhmr/go-common/either"
)

func TestSize(t *testing.T) {
	if size := reflect.TypeFor[either.Either[any, any]]().Size(); size != 16 {
		t.Errorf("Either should only consume 16 bytes, got %d bytes!", size)
	}
}

func TestEitherNeither(t *testing.T) {
	o := either.Either[struct{}, struct{}]{}
	if _, exists := o.Left(); exists {
		t.Errorf("Left() should not exist")
	}
	if _, exists := o.Right(); exists {
		t.Errorf("Right() should not exist")
	}
	if !o.Neither() {
		t.Errorf("Neither() should be true")
	}
}

func TestEitherLeftNil(t *testing.T) {
	o := either.Left[any, any](nil)
	if value, exists := o.Left(); !exists {
		t.Errorf("Left() should exist")
	} else if value != nil {
		t.Errorf("Left() wrong value")
	}
	if _, exists := o.Right(); exists {
		t.Errorf("Right() should not exist")
	}
	if o.Neither() {
		t.Errorf("Neither() should be false")
	}
}

func TestEitherRightNil(t *testing.T) {
	o := either.Right[any, any](nil)
	if _, exists := o.Left(); exists {
		t.Errorf("Left() should not exist")
	}
	if value, exists := o.Right(); !exists {
		t.Errorf("Right() should exist")
	} else if value != nil {
		t.Errorf("Right() wrong value")
	}
	if o.Neither() {
		t.Errorf("Neither() should be false")
	}
}

func TestEitherLeftByte(t *testing.T) {
	o := either.Left[byte, byte](7)
	if value, exists := o.Left(); !exists {
		t.Errorf("Left() should exist")
	} else if value != byte(7) {
		t.Errorf("Left() wrong value")
	}
	if _, exists := o.Right(); exists {
		t.Errorf("Right() should not exist")
	}
	if o.Neither() {
		t.Errorf("Neither() should be false")
	}
}

func TestEitherRightByte(t *testing.T) {
	o := either.Right[byte, byte](7)
	if _, exists := o.Left(); exists {
		t.Errorf("Left() should not exist")
	}
	if value, exists := o.Right(); !exists {
		t.Errorf("Right() should exist")
	} else if value != byte(7) {
		t.Errorf("Right() wrong value")
	}
	if o.Neither() {
		t.Errorf("Neither() should be false")
	}
}
