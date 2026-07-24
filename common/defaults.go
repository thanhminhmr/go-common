/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package common

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/thanhminhmr/go-exception"
)

// ApplyDefaults applies "default" struct tag values to the exported fields of
// the struct pointed to by v, recursing into nested struct fields (direct,
// pointer, or anonymous embedded). A nil pointer field is allocated only if its
// subtree contains at least one "default" tag.
//
// The "default" tag is honored on built-in scalar types and on types
// implementing [mapstructure.Unmarshaler] or [encoding.TextUnmarshaler], via
// [BindSimpleFromString].
//
// ApplyDefaults panics if value is not a (non-nil) pointer to a struct. It
// returns an error if a "default" tag value cannot be bound to its field, or
// if the struct nesting is too deep.
func ApplyDefaults(value any) error {
	if reflectValue := reflect.ValueOf(value); reflectValue.Kind() == reflect.Pointer {
		if reflectValue := reflectValue.Elem(); reflectValue.Kind() == reflect.Struct {
			return applyPlan(reflectValue, getDefaultsPlan(reflectValue.Type()), 0)
		}
	}
	panic(fmt.Sprintf("ApplyDefaults: v must be a pointer to struct, got %T", value))
}

type defaultPlan struct {
	nodes []defaultNode
	done  bool
}

type defaultNode = struct {
	index int
	value string
	inner *defaultPlan
}

var defaultsCache sync.Map // map[reflect.Type]*[]defaultNode

func getDefaultsPlan(t reflect.Type) *defaultPlan {
	var plan defaultPlan
	if built, exists := defaultsCache.LoadOrStore(t, &plan); exists {
		return built.(*defaultPlan)
	}
	buildDefaultsPlan(t, &plan)
	return &plan
}

func buildDefaultsPlan(t reflect.Type, plan *defaultPlan) {
	for index := range t.NumField() {
		field := t.Field(index)
		if !field.IsExported() {
			continue
		}
		if value, exists := field.Tag.Lookup("default"); exists {
			plan.nodes = append(plan.nodes, defaultNode{index: index, value: value})
			continue
		}
		fieldType := field.Type
		for fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}
		if fieldType.Kind() == reflect.Struct {
			if innerPlan := getDefaultsPlan(fieldType); !innerPlan.done || len(innerPlan.nodes) > 0 {
				plan.nodes = append(plan.nodes, defaultNode{index: index, inner: innerPlan})
			}
		}
	}
	plan.done = true
}

func applyPlan(v reflect.Value, plan *defaultPlan, depth uint) error {
	const maxDepth = 16
	for _, node := range plan.nodes {
		fieldValue := v.Field(node.index)
		for fieldValue.Kind() == reflect.Pointer {
			if fieldValue.IsNil() {
				fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
			}
			fieldValue = fieldValue.Elem()
		}
		if node.inner != nil {
			if depth >= maxDepth {
				return exception.String("ApplyDefaults: Struct too deep")
			} else if err := applyPlan(fieldValue, node.inner, depth+1); err != nil {
				return err
			}
		} else if err := BindSimpleFromString(node.value, fieldValue.Addr().Interface()); err != nil {
			return exception.Template("ApplyDefaults: Invalid default for %T.%s: %w").
				Format(v.Interface(), v.Type().Field(node.index).Name, err)
		}
	}
	return nil
}
