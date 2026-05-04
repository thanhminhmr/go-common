/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

// Package configuration provides a loader for values from `.env` files and
// environment variables.
//
// Values are resolved in increasing order of priority (later overrides earlier):
//  1. Struct `default` tag
//  2. [configuration.SetDefault] (no prefix)
//  3. `.env` file in the current directory (no prefix)
//  4. Environment variables (no prefix)
//
// This order is then repeated for each prefix level, applied from the last
// prefix backward (e.g., last prefix, last two prefixes, etc.).
package configuration

import (
	"os"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"
)

var globalConfigs = map[string]string{}

func init() {
	// os.Environ() have the highest priority
	globalParseLines(os.Environ())

	// .env file in the current directory have higher priority than defaults
	bytes, err := os.ReadFile(".env")
	if err == nil {
		globalParseLines(strings.Split(string(bytes), "\n"))
	}
}

func globalParseLines(lines []string) {
	for _, line := range lines {
		split := strings.SplitN(line, "=", 2)
		if len(split) == 2 {
			SetDefault(split[0], split[1])
		}
	}
}

func SetDefault(key string, value string) {
	key = strings.TrimSpace(key)
	if _, exists := globalConfigs[key]; !exists {
		globalConfigs[key] = strings.TrimSpace(value)
	}
}

// Load creates and fills a configuration struct from multiple sources. Check the
// package documents for the configuration priority.
func Load[Type any]() (*Type, error) {
	// create empty new struct and keys
	config, configKeys := createConfigStructAndKeys[Type]()
	if err := loadWithPrefix(config, configKeys); err != nil {
		return nil, err
	}
	return config, nil
}

// Loader creates and fills a configuration struct from multiple sources. Check
// the package documents for the configuration priority.
func Loader[Type any](prefixes ...string) func() (*Type, error) {
	return func() (*Type, error) {
		// create empty new struct and keys
		config, configKeys := createConfigStructAndKeys[Type]()
		if err := loadWithPrefix(config, configKeys, prefixes...); err != nil {
			return nil, err
		}
		return config, nil
	}
}

// Load creates and fills a configuration struct from multiple sources. Check
// the package documents for the configuration priority.
func loadWithPrefix(config any, configKeys map[string]*string, prefixes ...string) error {
	// load config map from global
	configMap := loadConfigMapWithPrefixesFromGlobal(configKeys, prefixes)
	// create decoder
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:          "env",
		DecodeHook:       internalDecodeHookFunc,
		ZeroFields:       true,
		WeaklyTypedInput: true,
		Result:           config,
	})
	if err != nil {
		panic("BUG: Decoder must be valid: " + err.Error())
	}
	// decode and validate
	if err := decoder.Decode(configMap); err != nil {
		return err
	} else if err := internalValidator.Struct(config); err != nil {
		return err
	}
	return nil
}

func createConfigStructAndKeys[Type any]() (*Type, map[string]*string) {
	// check if type is struct
	configType := reflect.TypeFor[Type]()
	if configType.Kind() != reflect.Struct {
		panic("BUG: Config type must be a struct")
	}
	// scan for keys and default values
	configKeys := make(map[string]*string, configType.NumField())
	for i := range configType.NumField() {
		configField := configType.Field(i)
		if key, ok := configField.Tag.Lookup("env"); ok {
			if value, ok := configField.Tag.Lookup("default"); ok {
				configKeys[key] = &value
			} else {
				configKeys[key] = nil
			}
		}
	}
	// create empty new struct
	config := reflect.New(configType).Interface().(*Type)
	return config, configKeys
}

func loadConfigMapWithPrefixesFromGlobal(configKeys map[string]*string, prefixes []string) map[string]string {
	// create new empty map
	configMap := make(map[string]string, len(configKeys))
	// join all prefixes
	joinedPrefixes := make([]string, len(prefixes)+1)
	{
		prefixBuilder := ""
		for i, prefix := range slices.Backward(prefixes) {
			prefixBuilder = prefix + "_" + prefixBuilder
			joinedPrefixes[i] = prefixBuilder
		}
	}
	// scan global configs with prefixes
	for _, joinedPrefix := range joinedPrefixes {
		for globalKey, value := range globalConfigs {
			if cutKey, exists := strings.CutPrefix(globalKey, joinedPrefix); exists {
				if _, exists := configKeys[cutKey]; !exists {
					continue
				} else if _, exists := configMap[cutKey]; exists {
					continue
				}
				configMap[cutKey] = value
				continue
			}
		}
	}
	// scan left-over struct keys for default value
	for key, value := range configKeys {
		if value == nil {
			continue
		}
		if _, exists := configMap[key]; !exists {
			configMap[key] = *value
		}
	}
	return configMap
}

var internalValidator = validator.New(validator.WithRequiredStructEnabled())

var internalDecodeHookFunc = mapstructure.ComposeDecodeHookFunc(
	splitValueBySemicolonsIfTargetIsSlice,
	mapstructure.TextUnmarshallerHookFunc(),
	mapstructure.StringToBasicTypeHookFunc(),
	mapstructure.StringToTimeHookFunc(time.RFC3339Nano),
	mapstructure.StringToURLHookFunc(),
	mapstructure.StringToIPHookFunc(),
	mapstructure.StringToIPNetHookFunc(),
	mapstructure.StringToNetIPAddrHookFunc(),
	mapstructure.StringToNetIPAddrPortHookFunc(),
	mapstructure.StringToNetIPPrefixHookFunc(),
	unboxIfElementSliceHasSingleElement,
)

func unboxIfElementSliceHasSingleElement(from reflect.Value, to reflect.Value) (any, error) {
	// convert single value slice to value
	if from.Kind() == reflect.Slice && from.Len() == 1 {
		toType := to.Type()
		for toType.Kind() == reflect.Ptr {
			toType = toType.Elem()
		}
		if toType.Kind() != reflect.Slice {
			return from.Index(0).Interface(), nil
		}
	}
	return from.Interface(), nil
}

func splitValueBySemicolonsIfTargetIsSlice(from reflect.Type, to reflect.Type, data any) (any, error) {
	if from.Kind() != reflect.String {
		return data, nil
	}
	if to.Kind() != reflect.Slice {
		return data, nil
	}
	raw := data.(string)
	if raw == "" {
		return []string{}, nil
	}
	return strings.Split(raw, ";"), nil
}
