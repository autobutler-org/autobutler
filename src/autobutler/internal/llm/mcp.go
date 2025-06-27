package llm

import (
	"autobutler/pkg/util"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/openai/openai-go"
)

var (
	mcpRegistry = &McpRegistry{
		Functions: make(map[string]McpFunction),
	}
)

type AddParams struct {
	Param0 float64 `json:"param0"`
	Param1 float64 `json:"param1"`
}

func (r McpRegistry) Add(param0 float64, param1 float64) float64 {
	return param0 + param1
}

type QueryInventoryParams struct {
	// Item to query for
	Param0 string `json:"param0"`
}

type QueryInventoryResponse struct {
	Item 	string  `json:"item"`
	Inventory float64 `json:"inventory"`
	Unit string `json:"unit,omitempty"`
}

func (r McpRegistry) QueryInventory(param0 string) QueryInventoryResponse {
	return QueryInventoryResponse{
		Item:      param0, // Example static response
		Inventory: 100.0, // Example static response
		Unit:      "gallons", // Example static unit
	}
}


func unmarshalParamSchema[T any](paramSchema string) (*T, error) {
	var params T
	if err := json.Unmarshal([]byte(paramSchema), &params); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
	}
	return &params, nil
}

func init() {
	// Add function
	addFn, err := NewMcpFunction(mcpRegistry.Add, "Adds two numbers together and returns the result.", func(result any, paramSchema string) (string, error) {
		parameters, err := unmarshalParamSchema[AddParams](paramSchema)
		if err != nil {
			return "", fmt.Errorf("failed to unmarshal parameters: %w", err)
		}
		return fmt.Sprintf("%f + %f = %f", parameters.Param0, parameters.Param1, result), nil
	})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for Add function: %v", err))
	}
	mcpRegistry.Functions[addFn.Name()] = *addFn

	// QueryInventory function
	queryInventoryFn, err := NewMcpFunction(mcpRegistry.QueryInventory, "Queries the home inventory for amount of an item.", func(result any, paramSchema string) (string, error) {
		parameters, err := unmarshalParamSchema[QueryInventoryParams](paramSchema)
		if err != nil {
			return "", fmt.Errorf("failed to unmarshal parameters: %w", err)
		}
		response := result.(QueryInventoryResponse)
		return fmt.Sprintf("There are %f %s of %s", response.Inventory, response.Unit, parameters.Param0), nil
	})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for QueryInventory function: %v", err))
	}
	mcpRegistry.Functions[queryInventoryFn.Name()] = *queryInventoryFn
}

type McpRegistry struct {
	Functions map[string]McpFunction
}

type McpFunction struct {
	Definition    openai.FunctionDefinitionParam
	fn            interface{}
	OutputHandler func(result any, paramSchema string) (string, error)
}

func (f McpFunction) Name() string {
	return f.Definition.Name
}

func (f McpFunction) Parameters() openai.FunctionParameters {
	return f.Definition.Parameters
}

func (f McpFunction) Description() string {
	return f.Definition.Description.String()
}

func NewMcpFunction(fn interface{}, description string, outputHandler func(result any, paramSchema string) (string, error)) (*McpFunction, error) {
	t := reflect.TypeOf(fn)
	if t.Kind() != reflect.Func {
		return nil, fmt.Errorf("expected a function, got %s", t.Kind())
	}

	params := map[string]any{}
	required := []string{}
	for i := range t.NumIn() {
		paramType := t.In(i)
		paramName := fmt.Sprintf("param%d", i)
		params[paramName] = util.TypeToJsonschema(paramType)
		required = append(required, paramName)
	}
	schema := map[string]any{
		"type":       "object",
		"properties": params,
		"required":   required,
	}

	return &McpFunction{
		Definition: openai.FunctionDefinitionParam{
			Name:        util.GetFunctionName(fn),
			Strict:      openai.Bool(false),
			Description: openai.String(description),
			Parameters:  schema,
		},
		fn:            fn,
		OutputHandler: outputHandler,
	}, nil
}

func (r McpRegistry) makeToolCall(toolCall openai.ChatCompletionMessageToolCall) (any, error) {
	var args map[string]any
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to unmarshal function arguments: %w", err)
	}
	if _, ok := r.Functions[toolCall.Function.Name]; !ok {
		return nil, fmt.Errorf("function %s not found in registry", toolCall.Function.Name)
	}
	fnDef, ok := r.Functions[toolCall.Function.Name]
	if !ok {
		return nil, fmt.Errorf("function %s not found in registry", toolCall.Function.Name)
	}

	// Prepare argument list in order as defined in fnDef.Parameters
	var paramNames []string
	if fnDef.Parameters() != nil {
		if props, ok := fnDef.Parameters()["properties"].(map[string]interface{}); ok {
			for name := range props {
				paramNames = append(paramNames, name)
			}
		}
	}

	var argValues []interface{}
	for _, name := range paramNames {
		val, exists := args[name]
		if !exists {
			return nil, fmt.Errorf("missing argument '%s' for function %s", name, toolCall.Function.Name)
		}
		argValues = append(argValues, val)
	}

	var returnValue any
	var err error
	if returnValue, err = r.callByName(toolCall.Function.Name, argValues...); err != nil {
		return nil, fmt.Errorf("failed to call function %s: %w", toolCall.Function.Name, err)
	}
	return returnValue, nil
}

func (r McpRegistry) callByName(fnName string, args ...any) (any, error) {
	fn := reflect.ValueOf(&r).MethodByName(fnName)
	if fn.Kind() != reflect.Func {
		return nil, fmt.Errorf("function %s not found", fnName)
	}

	if fn.Type().NumIn() != len(args) {
		return nil, fmt.Errorf("function %s expects %d arguments, got %d", fnName, fn.Type().NumIn(), len(args))
	}

	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		in[i] = reflect.ValueOf(arg)
	}

	out := fn.Call(in)
	if len(out) == 0 {
		return nil, nil // No return value
	}
	return out[0].Interface(), nil
}

func (r McpRegistry) toCompletionToolParam() []openai.ChatCompletionToolParam {
	var tools []openai.ChatCompletionToolParam
	for _, fn := range r.Functions {
		tools = append(tools, openai.ChatCompletionToolParam{
			Type:     "function",
			Function: fn.Definition,
		})
	}
	return tools
}
