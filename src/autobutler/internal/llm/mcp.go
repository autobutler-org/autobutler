package llm

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/openai/openai-go"
)

// GenerateJSONSchema generates a JSON Schema for the given function's parameters and returns an openai.FunctionDefinitionParam.
func GenerateJSONSchema(fn interface{}, description string) (*openai.FunctionDefinitionParam, error) {
	t := reflect.TypeOf(fn)
	if t.Kind() != reflect.Func {
		return nil, fmt.Errorf("expected a function, got %s", t.Kind())
	}

	params := map[string]any{}
	required := []string{}
	for i := 0; i < t.NumIn(); i++ {
		paramType := t.In(i)
		paramName := fmt.Sprintf("param%d", i)
		params[paramName] = typeToSchema(paramType)
		required = append(required, paramName)
	}
	schema := map[string]any{
		"type":       "object",
		"properties": params,
		"required":   required,
	}

	return &openai.FunctionDefinitionParam{
		Name:        getFunctionName(fn),
		Strict:      openai.Bool(false),
		Description: openai.String(description),
		Parameters:  schema,
	}, nil
}

func getFunctionName(fn interface{}) string {
    strs := strings.Split((runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()), ".")
    return strs[len(strs)-1]
}

func typeToSchema(t reflect.Type) map[string]any {
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
			"items": typeToSchema(t.Elem()),
		}
	case reflect.Map:
		return map[string]any{
			"type":                 "object",
			"additionalProperties": typeToSchema(t.Elem()),
		}
	case reflect.Struct:
		props := map[string]any{}
		required := []string{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			props[f.Name] = typeToSchema(f.Type)
			required = append(required, f.Name)
		}
		return map[string]any{
			"type":       "object",
			"properties": props,
			"required":   required,
		}
	case reflect.Ptr:
		return typeToSchema(t.Elem())
	default:
		return map[string]any{"type": "string"}
	}
}
