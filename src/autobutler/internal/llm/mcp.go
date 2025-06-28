package llm

import (
	"autobutler/internal/db"
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
	X float64 `json:"param0"`
	Y float64 `json:"param1"`
}

func (r McpRegistry) Add(x float64, y float64) float64 {
	return x + y
}

type QueryInventoryParams struct {
	ItemName string `json:"param0"`
}

type QueryInventoryResponse struct {
	Item 	string  `json:"item"`
	Inventory float64 `json:"inventory"`
	Unit string `json:"unit,omitempty"`
}

func (r McpRegistry) QueryInventory(itemName string) QueryInventoryResponse {
	item, err := db.Instance.QueryInventory(itemName)
	if err != nil {
		panic(fmt.Sprintf("failed to query inventory for item %s: %v", itemName, err))
	}
	if item == nil {
		return QueryInventoryResponse{
			Item:      itemName,
			Inventory: 0.0,
			Unit:      "",
		}
	}
	return QueryInventoryResponse{
		Item:      item.Name,
		Inventory: float64(item.Amount),
		Unit:      item.Unit,
	}
}

type AddToInventoryParams struct {
    Name   string `json:"param0"`
    Amount float64 `json:"param1"`
    Unit   string `json:"param2"`
}

type AddToInventoryResponse struct {
	Item 	db.Item  `json:"item"`
}

func (r McpRegistry) AddToInventory(name string, amount float64, unit string) AddToInventoryResponse {
	item, err := db.Instance.AddToInventory(db.Item{
		Name:   name,
		Amount: amount,
		Unit:   unit,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to add to inventory the item: %v", err))
	}
	return AddToInventoryResponse{
		Item: *item,
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
	// QueryInventory function
	queryInventoryFn, err := NewMcpFunction(mcpRegistry.QueryInventory, "Queries the home inventory for amount of an item.", func(result any, paramSchema string) (string, error) {
		parameters, err := unmarshalParamSchema[QueryInventoryParams](paramSchema)
		if err != nil {
			return "", fmt.Errorf("failed to unmarshal parameters: %w", err)
		}
		response := result.(QueryInventoryResponse)
		return fmt.Sprintf("There are %f %s of %s", response.Inventory, response.Unit, parameters.ItemName), nil
	})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for QueryInventory function: %v", err))
	}
	mcpRegistry.Functions[queryInventoryFn.Name()] = *queryInventoryFn

	// AddToInventory function
	addToInventoryFn, err := NewMcpFunction(mcpRegistry.AddToInventory, "Adds an item to the home inventory.", func(result any, paramSchema string) (string, error) {
		parameters, err := unmarshalParamSchema[AddToInventoryParams](paramSchema)
		if err != nil {
			return "", fmt.Errorf("failed to unmarshal parameters: %w", err)
		}
		response := result.(AddToInventoryResponse)
		return fmt.Sprintf("Added %f %s of %s to the inventory, so now you have %f %s.", parameters.Amount, response.Item.Unit, response.Item.Name, response.Item.Amount, response.Item.Unit), nil
	})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for AddToInventory function: %v", err))
	}
	mcpRegistry.Functions[addToInventoryFn.Name()] = *addToInventoryFn
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
