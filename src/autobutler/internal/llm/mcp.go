package llm

import (
	"autobutler/pkg/util"
	"fmt"
	"reflect"
	"strings"

	"github.com/openai/openai-go"
)

var (
	mcpRegistry = &McpRegistry{
		Functions: make(map[string]openai.FunctionDefinitionParam),
	}
)

func (r McpRegistry) Add(param0 float64, param1 float64) float64 {
	return param0 + param1
}

func init() {
	addFn, err := util.GenerateJSONSchema(mcpRegistry.Add, "Adds two numbers together and returns the result.")
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for add function: %v", err))
	}
	mcpRegistry.Functions[addFn.Name] = *addFn
}
