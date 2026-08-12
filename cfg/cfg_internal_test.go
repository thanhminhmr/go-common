/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package cfg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SetGlobalForTest replaces the global config for testing and returns a
// cleanup function that restores the previous value. Test-only.
func SetGlobalForTest(config map[string]any) {
	globalConfig = config
}

// =========================================================================
// deepMergeInto
// =========================================================================

func TestDeepMergeInto(t *testing.T) {
	t.Run("scalar replaces scalar", func(t *testing.T) {
		dst := map[string]any{"k": "old"}
		deepMergeInto(dst, map[string]any{"k": "new"})
		assert.Equal(t, "new", dst["k"])
	})

	t.Run("missing key added", func(t *testing.T) {
		dst := map[string]any{}
		deepMergeInto(dst, map[string]any{"k": "v"})
		assert.Equal(t, "v", dst["k"])
	})

	t.Run("objects merge recursively", func(t *testing.T) {
		dst := map[string]any{"o": map[string]any{"a": 1, "b": 2}}
		deepMergeInto(dst, map[string]any{"o": map[string]any{"b": 3, "c": 4}})
		assert.Equal(t, map[string]any{"a": 1, "b": 3, "c": 4}, dst["o"])
	})

	t.Run("nested objects merge deeply", func(t *testing.T) {
		dst := map[string]any{"o": map[string]any{"sub": map[string]any{"a": 1}}}
		deepMergeInto(dst, map[string]any{"o": map[string]any{"sub": map[string]any{"b": 2}}})
		assert.Equal(t, map[string]any{"sub": map[string]any{"a": 1, "b": 2}}, dst["o"])
	})

	t.Run("array replaces array (no concat)", func(t *testing.T) {
		dst := map[string]any{"a": []any{1, 2}}
		deepMergeInto(dst, map[string]any{"a": []any{3}})
		assert.Equal(t, []any{3}, dst["a"])
	})

	t.Run("array replaces object", func(t *testing.T) {
		dst := map[string]any{"k": map[string]any{"a": 1}}
		deepMergeInto(dst, map[string]any{"k": []any{1, 2}})
		assert.Equal(t, []any{1, 2}, dst["k"])
	})

	t.Run("null replaces value", func(t *testing.T) {
		dst := map[string]any{"k": "v"}
		deepMergeInto(dst, map[string]any{"k": nil})
		_, exists := dst["k"]
		assert.True(t, exists, "null should be stored as a value (key present)")
		assert.Nil(t, dst["k"])
	})

	t.Run("null replaces object", func(t *testing.T) {
		dst := map[string]any{"k": map[string]any{"a": 1}}
		deepMergeInto(dst, map[string]any{"k": nil})
		assert.Nil(t, dst["k"])
	})

	t.Run("scalar replaces object", func(t *testing.T) {
		dst := map[string]any{"k": map[string]any{"a": 1}}
		deepMergeInto(dst, map[string]any{"k": "s"})
		assert.Equal(t, "s", dst["k"])
	})

	t.Run("object replaces scalar", func(t *testing.T) {
		dst := map[string]any{"k": "s"}
		deepMergeInto(dst, map[string]any{"k": map[string]any{"a": 1}})
		assert.Equal(t, map[string]any{"a": 1}, dst["k"])
	})

	t.Run("key only in src added", func(t *testing.T) {
		dst := map[string]any{"a": 1}
		deepMergeInto(dst, map[string]any{"b": 2})
		assert.Equal(t, 1, dst["a"])
		assert.Equal(t, 2, dst["b"])
	})

	t.Run("empty src leaves dst unchanged", func(t *testing.T) {
		dst := map[string]any{"a": 1}
		deepMergeInto(dst, map[string]any{})
		assert.Equal(t, 1, dst["a"])
		assert.Len(t, dst, 1)
	})
}

// =========================================================================
// collectAndMerge
// =========================================================================

func TestCollectAndMerge(t *testing.T) {
	root := map[string]any{
		"db": map[string]any{"host": "h0", "port": 5432},
		"prod": map[string]any{
			"db":      map[string]any{"host": "hp"},
			"cluster": map[string]any{"db": map[string]any{"port": 6543}},
		},
	}

	t.Run("no prefix uses root level", func(t *testing.T) {
		m := collectAndMerge(root, "db", nil)
		assert.Equal(t, "h0", m["host"])
		assert.Equal(t, 5432, m["port"])
	})

	t.Run("single prefix overrides", func(t *testing.T) {
		m := collectAndMerge(root, "db", []string{"prod"})
		assert.Equal(t, "hp", m["host"])
		assert.Equal(t, 5432, m["port"])
	})

	t.Run("multiple prefixes deeper wins", func(t *testing.T) {
		m := collectAndMerge(root, "db", []string{"prod", "cluster"})
		assert.Equal(t, "hp", m["host"])
		assert.Equal(t, 6543, m["port"])
	})

	t.Run("missing prefix segment skips deeper layers", func(t *testing.T) {
		m := collectAndMerge(root, "db", []string{"staging", "cluster"})
		assert.Equal(t, "h0", m["host"])
		assert.Equal(t, 5432, m["port"])
	})

	t.Run("missing name at root yields empty", func(t *testing.T) {
		m := collectAndMerge(root, "cache", nil)
		assert.Empty(t, m)
	})

	t.Run("non-object intermediate skips deeper layers", func(t *testing.T) {
		r := map[string]any{
			"prod": "notobj",
			"db":   map[string]any{"host": "h0"},
		}
		m := collectAndMerge(r, "db", []string{"prod", "x"})
		assert.Equal(t, "h0", m["host"])
	})

	t.Run("non-object leaf skipped", func(t *testing.T) {
		r := map[string]any{"db": "notobj"}
		m := collectAndMerge(r, "db", nil)
		assert.Empty(t, m)
	})

	t.Run("empty root yields empty", func(t *testing.T) {
		m := collectAndMerge(map[string]any{}, "db", nil)
		assert.Empty(t, m)
	})

	t.Run("name exists at multiple levels all merged", func(t *testing.T) {
		r := map[string]any{
			"db":   map[string]any{"a": 1},
			"prod": map[string]any{"db": map[string]any{"b": 2}},
		}
		m := collectAndMerge(r, "db", []string{"prod"})
		assert.Equal(t, 1, m["a"])
		assert.Equal(t, 2, m["b"])
	})

	t.Run("name only in prefixed level initializes merged", func(t *testing.T) {
		// top-level "db" is missing so merged stays nil until the prefix
		// level provides the value (the "if merged == nil" branch).
		r := map[string]any{
			"prod": map[string]any{"db": map[string]any{"x": 1}},
		}
		m := collectAndMerge(r, "db", []string{"prod"})
		assert.Equal(t, map[string]any{"x": 1}, m)
	})
}

// =========================================================================
// loadGlobalConfig
// =========================================================================

func TestLoadGlobalConfigFromEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"app":{"name":"hi"}}`), 0o644))
	t.Setenv("CFG_FILE", path)

	config := loadGlobalConfig()
	app, ok := config["app"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "hi", app["name"])
}

func TestLoadGlobalConfigExplicitMissingFilePanics(t *testing.T) {
	t.Setenv("CFG_FILE", filepath.Join(t.TempDir(), "nope.json"))
	assert.Panics(t, func() {
		loadGlobalConfig()
	})
}

func TestLoadGlobalConfigMalformedJSONPanics(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{invalid`), 0o644))
	t.Setenv("CFG_FILE", path)
	assert.Panics(t, func() {
		loadGlobalConfig()
	})
}
