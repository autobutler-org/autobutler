package mcp

import (
	"autobutler/pkg/util"
	"fmt"
)

var (
	Registry = &McpRegistry{
		Functions: make(map[string]McpFunction),
	}
)

func init() {
	Register[QueryInventoryParams](Registry, Registry.QueryInventory, "Queries the home inventory for an item")
	Register[UpsertItemParams](Registry, Registry.UpsertItem, "Adds an item to the home inventory.")
	Register[ReduceInventoryParams](Registry, Registry.ReduceInventory, "Removes an item from the home inventory, such as when the user used some of the item.")
}

func Register[TParams Params](r *McpRegistry, fn any, description string) {
	function, err := NewMcpFunction(
		fn,
		description,
		func(result any, paramSchema string) (string, error) {
			parameters, err := util.UnmarshalParamSchema[TParams](paramSchema)
			if err != nil {
				return "", fmt.Errorf("failed to unmarshal parameters (%s): %w", paramSchema, err)
			}
			outputFmt, outputArgs := (*parameters).Output(result)
			return fmt.Sprintf(outputFmt, outputArgs...), nil
		},
	)
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for %s function: %v", function.Name(), err))
	}
	Registry.Functions[function.Name()] = *function
}
