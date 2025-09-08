package configuration

import (
	"os"
	"strings"

	"github.com/thanhminhmr/go-common/internal"

	"github.com/go-viper/mapstructure/v2"
)

var globalDefaults = make(map[string]string)
var globalEnvironments = make(map[string]string)

func init() {
	// .env file have higher priority than defaults
	bytes, err := os.ReadFile(".env")
	if err == nil {
		saveEnvironments(strings.Split(string(bytes), "\n"))
	}

	// os.Environ() have the highest priority
	saveEnvironments(os.Environ())
}

func saveEnvironments(lines []string) {
	for _, line := range lines {
		split := strings.SplitN(line, "=", 2)
		if len(split) == 2 {
			globalEnvironments[strings.TrimSpace(split[0])] = strings.TrimSpace(split[1])
		}
	}
}

func SetDefault(key string, value string) {
	globalDefaults[key] = value
}

func Load[T any](config *T, prefixes ...string) error {
	prefix := ""
	if len(prefixes) > 0 {
		prefix = strings.Join(prefixes, "_") + "_"
	}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:          "env",
		DecodeHook:       internal.SplitSemicolonsDecodeHookFunc,
		ZeroFields:       true,
		WeaklyTypedInput: true,
		Result:           config,
	})
	if err != nil {
		return err
	}
	if err := decoder.Decode(getEnvironment(prefix)); err != nil {
		return err
	}
	return internal.Validator.Struct(config)
}

func Loader[T any](config *T, prefixes ...string) func() (*T, error) {
	return func() (*T, error) {
		err := Load(config, prefixes...)
		return config, err
	}
}

func getEnvironment(prefix string) map[string]string {
	environments := make(map[string]string)
	for key, value := range globalDefaults {
		if fixedKey, hasPrefix := strings.CutPrefix(key, prefix); hasPrefix {
			environments[fixedKey] = value
		}
	}
	for key, value := range globalEnvironments {
		if fixedKey, hasPrefix := strings.CutPrefix(key, prefix); hasPrefix {
			environments[fixedKey] = value
		}
	}
	return environments
}
