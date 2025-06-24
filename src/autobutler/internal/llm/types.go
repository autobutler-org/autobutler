package llm

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/openai/openai-go"
)

type ChatRole string

const (
	ChatRoleUser   ChatRole = "user"
	ChatRoleSystem ChatRole = "system"
	ChatRoleDummy  ChatRole = "dummy"
	ChatRoleError  ChatRole = "error"
)

type ChatMessage struct {
	Role      ChatRole `json:"role"`
	Content   string   `json:"content"`
	Timestamp string   `json:"timestamp"` // ISO 8601 format
}

func ErrorChatMessage(err error) ChatMessage {
	return ChatMessage{
		Role:      ChatRoleError,
		Content:   fmt.Sprintf("An error occurred while processing your request: %v", err),
		Timestamp: GetTimestamp(time.Now()),
	}
}

func GetTimestamp(timestamp time.Time) string {
	// Matches JS new Date().toLocaleTimeString()
	return timestamp.Format("3:04:05 PM")
}

func FromCompletionToChatMessage(completion openai.ChatCompletion) ChatMessage {
	if len(completion.Choices) == 0 {
		return ChatMessage{
			Role:      ChatRoleError,
			Content:   "",
			Timestamp: GetTimestamp(time.Now()),
		}
	}
	return ChatMessage{
		Role:    ChatRoleSystem,
		Content: completion.Choices[0].Message.Content,
		// Matches JS new Date().toLocaleTimeString()
		Timestamp: GetTimestamp(time.Now()),
	}
}

type McpRegistry struct {
	Functions map[string]openai.FunctionDefinitionParam
}

func (r McpRegistry) MakeToolCall(completion *openai.ChatCompletion) ([]any, error) {
	toolCalls := completion.Choices[0].Message.ToolCalls
	results := make([]any, 0, len(toolCalls))
	if len(toolCalls) > 0 {
		for _, toolCall := range toolCalls {
			var args map[string]float64
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
			if fnDef.Parameters != nil {
				if props, ok := fnDef.Parameters["properties"].(map[string]interface{}); ok {
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
			results = append(results, returnValue)
		}
	}
	return results, nil
}

func (r McpRegistry) callByName(fnName string, args ...any) (any, error) {
	actualFnName := strings.TrimSuffix(fnName,"-fm")
	fn := reflect.ValueOf(&r).MethodByName(actualFnName)
	if fn.Kind() != reflect.Func {
		return nil, fmt.Errorf("function %s not found", actualFnName)
	}

	if fn.Type().NumIn() != len(args) {
		return nil, fmt.Errorf("function %s expects %d arguments, got %d", actualFnName, fn.Type().NumIn(), len(args))
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

func (r McpRegistry) toCompletionToolParam() []openai.ChatCompletionToolParam{
	var tools []openai.ChatCompletionToolParam
	for _, fn := range r.Functions {
		tools = append(tools, openai.ChatCompletionToolParam{
			Type:        "function",
			Function:    fn,
		})
	}
	return tools
}
