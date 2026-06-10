/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package ctrl

import (
	"reflect"
	"slices"
	"testing"

	"github.com/rs/zerolog"
)

func TestUnsafeZerologLoggerAccess(t *testing.T) {
	external := reflect.TypeFor[zerolog.Logger]()
	internal := reflect.TypeFor[zerologLogger]()
	if external.Size() != internal.Size() || external.NumField() != internal.NumField() {
		t.Error("unsafe: zerolog.Logger does not match zerologLogger struct")
		t.FailNow()
	}
	for i := range external.NumField() {
		externalField := external.Field(i)
		internalField := internal.Field(i)
		if externalField.Name != internalField.Name || externalField.Type != internalField.Type ||
			externalField.Offset != internalField.Offset || !slices.Equal(externalField.Index, internalField.Index) ||
			externalField.Anonymous != internalField.Anonymous {
			t.Error("unsafe: zerolog.Logger does not match zerologLogger struct")
			t.FailNow()
		}
	}
}

func TestUnsafeZerologContextAccess(t *testing.T) {
	type zerologContext struct {
		l zerolog.Logger
	}
	external := reflect.TypeFor[zerolog.Context]()
	internal := reflect.TypeFor[zerologContext]()
	if external.Size() != internal.Size() || external.NumField() != internal.NumField() {
		t.Error("unsafe: zerolog.Context does not match zerologContext struct")
		t.FailNow()
	}
	for i := range external.NumField() {
		externalField := external.Field(i)
		internalField := internal.Field(i)
		if externalField.Name != internalField.Name || externalField.Type != internalField.Type ||
			externalField.Offset != internalField.Offset || !slices.Equal(externalField.Index, internalField.Index) ||
			externalField.Anonymous != internalField.Anonymous {
			t.Error("unsafe: zerolog.Context does not match zerologContext struct")
			t.FailNow()
		}
	}
}
