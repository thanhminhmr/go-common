package internal

import (
	"reflect"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
)

var DefaultDecodeHookFunc = mapstructure.ComposeDecodeHookFunc(
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

var SplitSemicolonsDecodeHookFunc = mapstructure.ComposeDecodeHookFunc(
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
