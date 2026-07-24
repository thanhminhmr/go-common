/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

// Package cfg loads a JSON configuration file and binds a selected sub-object
// to a struct.
//
// The file path is taken from the CFG_FILE environment variable, defaulting to
// "config.json". If CFG_FILE is unset and the file does not exist, cfg starts
// with an empty configuration; if CFG_FILE is set explicitly and the file is
// missing or malformed, cfg panics during init.
//
// [Load] and [LoadInto] select a sub-object by a name combined with zero or
// more prefixes (outer to inner). At each nesting level a same-named object may
// exist; all are deep-merged from shallowest to deepest, with deeper levels
// taking priority. Objects merge recursively; other values replace.
//
// Binding uses the "cfg" struct tag (untagged fields are ignored). The
// "default" struct tag supplies values for absent scalar fields. The result is
// validated with [common.ValidateStruct].
//
// [Load] and [LoadInto] are safe for concurrent use.
package cfg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"reflect"

	"github.com/thanhminhmr/go-common/common"
)

var globalConfig = loadGlobalConfig()

func loadGlobalConfig() map[string]any {
	file, explicit := "config.json", false
	if fileFromEnv, exists := os.LookupEnv("CFG_FILE"); exists {
		file, explicit = fileFromEnv, true
	}
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) && !explicit {
			return map[string]any{}
		}
		panic(fmt.Sprintf(`cfg: cannot read config file "%q": %v`, file, err))
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var config map[string]any
	if err := decoder.Decode(&config); err != nil {
		panic(fmt.Sprintf(`cfg: malformed JSON in config file "%q": %v`, file, err))
	}
	return config
}

// Load creates and fills a configuration struct from the merged sub-objects at
// the path formed by prefixes (outer to inner) and name. See the package
// documentation for merge and priority semantics.
func Load[Type any](name string, prefixes ...string) (*Type, error) {
	var config Type
	if err := LoadInto(&config, name, prefixes...); err != nil {
		return nil, err
	}
	return &config, nil
}

// LoadInto fills a configuration struct from the merged sub-objects at the path
// formed by prefixes (outer to inner) and name. See the package documentation
// for merge and priority semantics.
func LoadInto[Type any](config *Type, name string, prefixes ...string) error {
	if reflect.TypeFor[Type]().Kind() != reflect.Struct {
		panic("cfg: Type must be a struct")
	}
	if name == "" {
		panic("cfg: name must be non-empty")
	}
	for _, p := range prefixes {
		if p == "" {
			panic("cfg: prefix must be non-empty")
		}
	}
	merged := collectAndMerge(globalConfig, name, prefixes)
	if err := common.ApplyDefaults(config); err != nil {
		return err
	} else if err := common.BindStructWithTag("cfg", merged, config); err != nil {
		return err
	}
	return common.ValidateStruct(config)
}

// collectAndMerge walks the global config object and collects all sub-objects
// found at the path formed by prefixes (outer to inner) followed by name. At
// each level — root, prefixes[0], prefixes[0][prefixes[1]], … — the same-named
// object may exist; the collected objects are deep-merged in order from
// shallowest to deepest, with deeper levels taking priority.
//
// A level contributes nothing if any path segment is missing or if an
// intermediate value is not a JSON object, or if the final value is not a JSON
// object.
func collectAndMerge(root map[string]any, name string, prefixes []string) map[string]any {
	var merged map[string]any
	readOnly := true
	// get top level value
	if value, exists := root[name]; exists {
		if value, ok := value.(map[string]any); ok {
			merged = value
		}
	}
	// for each prefix
	for _, prefix := range prefixes {
		// go down one layer
		switch leaf, exists := root[prefix]; exists {
		case true:
			if value, ok := leaf.(map[string]any); ok {
				root = value
				break
			}
			fallthrough
		case false:
			return merged
		}
		// get this level value
		if value, exists := root[name]; exists {
			if value, ok := value.(map[string]any); ok {
				// if never found
				if merged == nil {
					merged = value
					continue
				}
				// if not cloned yet
				if readOnly {
					merged = maps.Clone(merged)
					readOnly = false
				}
				// deep merge
				deepMergeInto(merged, value)
			}
		}
	}
	return merged
}

// deepMergeInto merges src into dst in place. Object values are merged
// recursively; all other value types (scalars, arrays, null) replace the
// existing value.
func deepMergeInto(dst, src map[string]any) {
	for k, sv := range src {
		// If both dst and src hold objects at this key, merge recursively;
		// otherwise (scalars, arrays, null, or type mismatch) src replaces.
		if dv, exists := dst[k]; exists {
			if dm, ok := dv.(map[string]any); ok {
				if sm, ok := sv.(map[string]any); ok {
					deepMergeInto(dm, sm)
					continue
				}
			}
		}
		dst[k] = sv
	}
}
