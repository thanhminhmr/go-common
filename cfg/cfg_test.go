/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package cfg_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thanhminhmr/go-common/cfg"
)

// setGlobalJSON decodes jsonStr with UseNumber (matching production) and
// installs it as the global config for the duration of the test.
func setGlobalJSON(t *testing.T, jsonStr string) {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader([]byte(jsonStr)))
	dec.UseNumber()
	var m map[string]any
	require.NoError(t, dec.Decode(&m))
	cfg.SetGlobalForTest(m)
}

// =========================================================================
// Basic loading
// =========================================================================

func TestLoadBasicStruct(t *testing.T) {
	setGlobalJSON(t, `{"app":{"name":"myapp","port":8080}}`)
	type App struct {
		Name string `cfg:"name"`
		Port int    `cfg:"port"`
	}
	a, err := cfg.Load[App]("app")
	require.NoError(t, err)
	assert.Equal(t, "myapp", a.Name)
	assert.Equal(t, 8080, a.Port)
}

func TestLoadIntoFillsExisting(t *testing.T) {
	setGlobalJSON(t, `{"app":{"name":"x"}}`)
	type App struct {
		Name string `cfg:"name"`
	}
	var a App
	require.NoError(t, cfg.LoadInto(&a, "app"))
	assert.Equal(t, "x", a.Name)
}

func TestLoadReturnsDifferentPointers(t *testing.T) {
	setGlobalJSON(t, `{"app":{"name":"x"}}`)
	type App struct {
		Name string `cfg:"name"`
	}
	a, err := cfg.Load[App]("app")
	require.NoError(t, err)
	b, err := cfg.Load[App]("app")
	require.NoError(t, err)
	assert.NotSame(t, a, b)
}

func TestUntaggedFieldIgnored(t *testing.T) {
	setGlobalJSON(t, `{"app":{"name":"x","extra":"y"}}`)
	type App struct {
		Name  string `cfg:"name"`
		Extra string
	}
	a, err := cfg.Load[App]("app")
	require.NoError(t, err)
	assert.Equal(t, "x", a.Name)
	assert.Empty(t, a.Extra)
}

// =========================================================================
// Nested structs
// =========================================================================

func TestLoadNestedStruct(t *testing.T) {
	setGlobalJSON(t, `{"db":{"host":"h","port":5432,"creds":{"user":"u","pass":"p"}}}`)
	type DB struct {
		Host  string `cfg:"host"`
		Port  int    `cfg:"port"`
		Creds struct {
			User string `cfg:"user"`
			Pass string `cfg:"pass"`
		} `cfg:"creds"`
	}
	d, err := cfg.Load[DB]("db")
	require.NoError(t, err)
	assert.Equal(t, "h", d.Host)
	assert.Equal(t, 5432, d.Port)
	assert.Equal(t, "u", d.Creds.User)
	assert.Equal(t, "p", d.Creds.Pass)
}

func TestEmbeddedStruct(t *testing.T) {
	setGlobalJSON(t, `{"app":{"host":"h","port":80}}`)
	type Base struct {
		Host string `cfg:"host"`
	}
	type App struct {
		Base
		Port int `cfg:"port"`
	}
	a, err := cfg.Load[App]("app")
	require.NoError(t, err)
	assert.Equal(t, "h", a.Host)
	assert.Equal(t, 80, a.Port)
}

// =========================================================================
// Arrays
// =========================================================================

func TestLoadArrayToSlice(t *testing.T) {
	setGlobalJSON(t, `{"app":{"tags":["a","b","c"],"ports":[80,443]}}`)
	type App struct {
		Tags  []string `cfg:"tags"`
		Ports []int    `cfg:"ports"`
	}
	a, err := cfg.Load[App]("app")
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, a.Tags)
	assert.Equal(t, []int{80, 443}, a.Ports)
}

func TestLoadNestedArrayOfObjects(t *testing.T) {
	setGlobalJSON(t, `{"app":{"endpoints":[{"url":"u1"},{"url":"u2"}]}}`)
	type Endpoint struct {
		URL string `cfg:"url"`
	}
	type App struct {
		Endpoints []Endpoint `cfg:"endpoints"`
	}
	a, err := cfg.Load[App]("app")
	require.NoError(t, err)
	require.Len(t, a.Endpoints, 2)
	assert.Equal(t, "u1", a.Endpoints[0].URL)
	assert.Equal(t, "u2", a.Endpoints[1].URL)
}

// =========================================================================
// Scalar types
// =========================================================================

func TestLoadScalars(t *testing.T) {
	setGlobalJSON(t, `{"s":{"str":"hi","i":42,"u":7,"f":3.14,"b":true,"c":"1+2i"}}`)
	type S struct {
		Str string     `cfg:"str"`
		I   int        `cfg:"i"`
		U   uint       `cfg:"u"`
		F   float64    `cfg:"f"`
		B   bool       `cfg:"b"`
		C   complex128 `cfg:"c"`
	}
	s, err := cfg.Load[S]("s")
	require.NoError(t, err)
	assert.Equal(t, "hi", s.Str)
	assert.Equal(t, 42, s.I)
	assert.Equal(t, uint(7), s.U)
	assert.Equal(t, 3.14, s.F)
	assert.True(t, s.B)
	assert.Equal(t, 1+2i, s.C)
}

func TestLoadNegativeInt(t *testing.T) {
	setGlobalJSON(t, `{"s":{"v":-5}}`)
	type S struct {
		V int `cfg:"v"`
	}
	s, err := cfg.Load[S]("s")
	require.NoError(t, err)
	assert.Equal(t, -5, s.V)
}

// =========================================================================
// Prefix layering
// =========================================================================

func TestLoadPrefixLayering(t *testing.T) {
	setGlobalJSON(t, `{
		"db": {"host": "h0", "port": 5432},
		"prod": {"db": {"host": "hp"}, "cluster": {"db": {"port": 6543}}}
	}`)
	type DB struct {
		Host string `cfg:"host"`
		Port int    `cfg:"port"`
	}

	t.Run("no prefix", func(t *testing.T) {
		d, err := cfg.Load[DB]("db")
		require.NoError(t, err)
		assert.Equal(t, "h0", d.Host)
		assert.Equal(t, 5432, d.Port)
	})

	t.Run("single prefix overrides", func(t *testing.T) {
		d, err := cfg.Load[DB]("db", "prod")
		require.NoError(t, err)
		assert.Equal(t, "hp", d.Host)
		assert.Equal(t, 5432, d.Port)
	})

	t.Run("deeper prefix wins", func(t *testing.T) {
		d, err := cfg.Load[DB]("db", "prod", "cluster")
		require.NoError(t, err)
		assert.Equal(t, "hp", d.Host)
		assert.Equal(t, 6543, d.Port)
	})
}

func TestDeepMergeNestedObjectsAcrossLayers(t *testing.T) {
	setGlobalJSON(t, `{
		"db": {"creds": {"user": "u0", "pass": "p0"}, "host": "h0"},
		"prod": {"db": {"creds": {"user": "up"}}}
	}`)
	type DB struct {
		Host  string `cfg:"host"`
		Creds struct {
			User string `cfg:"user"`
			Pass string `cfg:"pass"`
		} `cfg:"creds"`
	}
	d, err := cfg.Load[DB]("db", "prod")
	require.NoError(t, err)
	assert.Equal(t, "h0", d.Host)
	assert.Equal(t, "up", d.Creds.User)
	assert.Equal(t, "p0", d.Creds.Pass)
}

func TestArrayReplacedInMerge(t *testing.T) {
	setGlobalJSON(t, `{
		"app": {"tags": ["a","b"]},
		"prod": {"app": {"tags": ["c"]}}
	}`)
	type App struct {
		Tags []string `cfg:"tags"`
	}
	a, err := cfg.Load[App]("app", "prod")
	require.NoError(t, err)
	assert.Equal(t, []string{"c"}, a.Tags)
}

// =========================================================================
// Defaults
// =========================================================================

func TestDefaultAppliedWhenAbsent(t *testing.T) {
	setGlobalJSON(t, `{"app":{}}`)
	type App struct {
		Name string `cfg:"name" default:"fallback"`
		Port int    `cfg:"port" default:"8080"`
	}
	a, err := cfg.Load[App]("app")
	require.NoError(t, err)
	assert.Equal(t, "fallback", a.Name)
	assert.Equal(t, 8080, a.Port)
}

func TestDefaultOverriddenByJSON(t *testing.T) {
	setGlobalJSON(t, `{"app":{"name":"real","port":9090}}`)
	type App struct {
		Name string `cfg:"name" default:"fallback"`
		Port int    `cfg:"port" default:"8080"`
	}
	a, err := cfg.Load[App]("app")
	require.NoError(t, err)
	assert.Equal(t, "real", a.Name)
	assert.Equal(t, 9090, a.Port)
}

func TestDefaultOnNestedStructAbsent(t *testing.T) {
	setGlobalJSON(t, `{}`)
	type App struct {
		DB struct {
			Host string `cfg:"host" default:"localhost"`
			Port int    `cfg:"port" default:"5432"`
		} `cfg:"db"`
	}
	a, err := cfg.Load[App]("app")
	require.NoError(t, err)
	assert.Equal(t, "localhost", a.DB.Host)
	assert.Equal(t, 5432, a.DB.Port)
}

func TestDefaultOnNestedStructPartialJSON(t *testing.T) {
	setGlobalJSON(t, `{"app":{"db":{"host":"remote"}}}`)
	type App struct {
		DB struct {
			Host string `cfg:"host" default:"localhost"`
			Port int    `cfg:"port" default:"5432"`
		} `cfg:"db"`
	}
	a, err := cfg.Load[App]("app")
	require.NoError(t, err)
	assert.Equal(t, "remote", a.DB.Host)
	assert.Equal(t, 5432, a.DB.Port)
}

func TestDefaultOnEmbeddedStruct(t *testing.T) {
	setGlobalJSON(t, `{}`)
	type Base struct {
		Host string `cfg:"host" default:"localhost"`
	}
	type App struct {
		Base
	}
	a, err := cfg.Load[App]("app")
	require.NoError(t, err)
	assert.Equal(t, "localhost", a.Host)
}

func TestDefaultBool(t *testing.T) {
	setGlobalJSON(t, `{"app":{}}`)
	type App struct {
		Verbose bool `cfg:"verbose" default:"true"`
		Quiet   bool `cfg:"quiet" default:"false"`
	}
	a, err := cfg.Load[App]("app")
	require.NoError(t, err)
	assert.True(t, a.Verbose)
	assert.False(t, a.Quiet)
}

func TestDefaultFloat(t *testing.T) {
	setGlobalJSON(t, `{"app":{}}`)
	type App struct {
		Score float64 `cfg:"score" default:"3.14"`
	}
	a, err := cfg.Load[App]("app")
	require.NoError(t, err)
	assert.Equal(t, 3.14, a.Score)
}

// =========================================================================
// Custom types via TextUnmarshaler
// =========================================================================

type customMode string

func (m *customMode) UnmarshalText(b []byte) error {
	*m = customMode(b)
	return nil
}

func TestLoadCustomTextUnmarshaler(t *testing.T) {
	setGlobalJSON(t, `{"app":{"mode":"auto"}}`)
	type App struct {
		Mode customMode `cfg:"mode"`
	}
	a, err := cfg.Load[App]("app")
	require.NoError(t, err)
	assert.Equal(t, customMode("auto"), a.Mode)
}

func TestDefaultOnCustomUnmarshaler(t *testing.T) {
	setGlobalJSON(t, `{"app":{}}`)
	type App struct {
		Mode customMode `cfg:"mode" default:"auto"`
	}
	a, err := cfg.Load[App]("app")
	require.NoError(t, err)
	assert.Equal(t, customMode("auto"), a.Mode)
}

// =========================================================================
// Validation
// =========================================================================

func TestValidationRequiredFails(t *testing.T) {
	setGlobalJSON(t, `{"app":{}}`)
	type App struct {
		Name string `cfg:"name" validate:"required"`
	}
	_, err := cfg.Load[App]("app")
	assert.Error(t, err)
}

func TestValidationRequiredPasses(t *testing.T) {
	setGlobalJSON(t, `{"app":{"name":"x"}}`)
	type App struct {
		Name string `cfg:"name" validate:"required"`
	}
	a, err := cfg.Load[App]("app")
	require.NoError(t, err)
	assert.Equal(t, "x", a.Name)
}

func TestValidationMinMax(t *testing.T) {
	setGlobalJSON(t, `{"app":{"level":99}}`)
	type App struct {
		Level int `cfg:"level" default:"5" validate:"min=-1,max=7"`
	}
	_, err := cfg.Load[App]("app")
	assert.Error(t, err)
}

// =========================================================================
// Missing config / empty global
// =========================================================================

func TestLoadFromEmptyGlobal(t *testing.T) {
	setGlobalJSON(t, `{}`)
	type App struct {
		Name string `cfg:"name" default:"def"`
	}
	a, err := cfg.Load[App]("app")
	require.NoError(t, err)
	assert.Equal(t, "def", a.Name)
}

func TestLoadMissingNameGetsDefaults(t *testing.T) {
	setGlobalJSON(t, `{"other":{}}`)
	type App struct {
		Name string `cfg:"name" default:"def"`
	}
	a, err := cfg.Load[App]("app")
	require.NoError(t, err)
	assert.Equal(t, "def", a.Name)
}

// =========================================================================
// Panics on misuse
// =========================================================================

func TestEmptyNamePanics(t *testing.T) {
	assert.Panics(t, func() {
		_, _ = cfg.Load[struct {
			X int `cfg:"x"`
		}]("")
	})
}

func TestEmptyPrefixPanics(t *testing.T) {
	assert.Panics(t, func() {
		_, _ = cfg.Load[struct {
			X int `cfg:"x"`
		}]("app", "")
	})
}

func TestNonStructTypePanics(t *testing.T) {
	assert.Panics(t, func() {
		_, _ = cfg.Load[int]("app")
	})
}

// =========================================================================
// String field from json.Number (UseNumber path)
// =========================================================================

func TestStringFromJSONNumber(t *testing.T) {
	setGlobalJSON(t, `{"app":{"port":8080}}`)
	type App struct {
		Port string `cfg:"port"`
	}
	a, err := cfg.Load[App]("app")
	require.NoError(t, err)
	assert.Equal(t, "8080", a.Port)
}
