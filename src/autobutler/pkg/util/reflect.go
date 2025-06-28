package util

import (
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

func GetFunctionName(fn interface{}) string {
	strs := strings.Split((runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()), ".")
	return strings.Split(strs[len(strs)-1], "-")[0]
}

func TypeToJsonschema(t reflect.Type) map[string]any {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Slice, reflect.Array:
		return map[string]any{
			"type":  "array",
			"items": TypeToJsonschema(t.Elem()),
		}
	case reflect.Map:
		return map[string]any{
			"type":                 "object",
			"additionalProperties": TypeToJsonschema(t.Elem()),
		}
	case reflect.Struct:
		props := map[string]any{}
		required := []string{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			props[f.Name] = TypeToJsonschema(f.Type)
			required = append(required, f.Name)
		}
		return map[string]any{
			"type":       "object",
			"properties": props,
			"required":   required,
		}
	case reflect.Ptr:
		return TypeToJsonschema(t.Elem())
	default:
		return map[string]any{"type": "string"}
	}
}

func UnmarshalParamSchema[T any](paramSchema string) (*T, error) {
	var params T
	if err := json.Unmarshal([]byte(paramSchema), &params); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
	}
	return &params, nil
}
