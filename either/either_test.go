/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package either_test

import (
	"testing"

	"github.com/thanhminhmr/go-common/either"
)

func TestEitherNeither(t *testing.T) {
	o := either.Either[struct{}, struct{}]{}
	if _, exists := o.Left(); exists {
		t.Errorf("Left() should not exist")
	}
	if _, exists := o.Right(); exists {
		t.Errorf("Right() should not exist")
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
}
