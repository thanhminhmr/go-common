/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package common

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaults_CacheReturnsSamePlan(t *testing.T) {
	type testType struct {
		A string `default:"x"`
		B int    `default:"1"`
	}
	typ := reflect.TypeFor[testType]()
	p1 := getDefaultsPlan(typ)
	p2 := getDefaultsPlan(typ)
	if assert.NotEmpty(t, p1.nodes) && assert.NotEmpty(t, p2.nodes) {
		assert.Same(t, &p1.nodes[0], &p2.nodes[0],
			"second call should return cached plan")
	}
}

func TestDefaults_PlanEmptyForNoDefaults(t *testing.T) {
	type noDefaults struct {
		A string
		B int
	}
	plan := getDefaultsPlan(reflect.TypeFor[noDefaults]())
	assert.Empty(t, plan.nodes)
}

func TestDefaults_PlanStructure(t *testing.T) {
	type inner struct {
		Host string `default:"localhost"`
	}
	type outer struct {
		Name  string `default:"app"`
		Inner inner
		Skip  string
	}
	plan := getDefaultsPlan(reflect.TypeFor[outer]())
	require.NotNil(t, plan)
	require.Len(t, plan.nodes, 2) // Name (leaf) + Inner (recurse); Skip excluded
	// First node: Name (leaf)
	assert.Equal(t, "app", plan.nodes[0].value)
	// Second node: Inner (recurse)
	require.NotNil(t, plan.nodes[1].inner)
	require.Len(t, plan.nodes[1].inner.nodes, 1)
	assert.Equal(t, "localhost", plan.nodes[1].inner.nodes[0].value)
}

func TestDefaults_SubPlanShared(t *testing.T) {
	type inner struct {
		Host string `default:"localhost"`
	}
	type parentA struct {
		Inner inner
	}
	type parentB struct {
		Inner inner
	}
	planA := getDefaultsPlan(reflect.TypeFor[parentA]())
	planB := getDefaultsPlan(reflect.TypeFor[parentB]())
	require.NotNil(t, planA)
	require.Len(t, planA.nodes, 1)
	require.NotNil(t, planB)
	require.Len(t, planB.nodes, 1)
	// Both parents should share the same cached sub-plan for inner
	assert.Same(t, planA.nodes[0].inner, planB.nodes[0].inner,
		"sub-plan should be shared between parent types")
}

func TestDefaults_RecursiveTypePlanStructure(t *testing.T) {
	type recursiveType struct {
		Val  int `default:"0"`
		Next *recursiveType
	}
	plan := getDefaultsPlan(reflect.TypeFor[recursiveType]())
	require.NotNil(t, plan)
	// Val (leaf) + Next (recurse, included via !done check during build)
	require.Len(t, plan.nodes, 2)
	assert.Equal(t, "0", plan.nodes[0].value)
	require.NotNil(t, plan.nodes[1].inner)
	assert.Same(t, plan, plan.nodes[1].inner,
		"recursive field's plan should be cyclic")
}
