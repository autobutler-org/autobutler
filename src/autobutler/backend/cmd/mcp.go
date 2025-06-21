package cmd

import (
	"encoding/json"

	"github.com/exokomodo/exoflow/autobutler/backend/internal/llm"
	"github.com/spf13/cobra"
)

var registry *llm.MCPRegistry

func init() {
	registry = llm.NewMCPRegistry()
	registry.RegisterFunction(
		&llm.Function{
			Name:       "list_functions",
			Parameters: []string{},
			Handler: func(params map[string]interface{}) (interface{}, error) {
				return registry.ListFunctions(), nil
			},
		},
	)
	registry.RegisterFunction(
		&llm.Function{
			Name:       "add",
			Parameters: []string{"x", "y"},
			Handler: func(params map[string]interface{}) (interface{}, error) {
				x := params["x"].(float64)
				y := params["y"].(float64)
				return x + y, nil
			},
		},
	)
}

func Mcp() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Test an mcp command directly with a JSON payload",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				cmd.Help()
				return
			}

			call := args[0]
			var command llm.MCPCommand
			if err := json.Unmarshal([]byte(call), &command); err != nil {
				cmd.PrintErrf("Invalid command JSON: %v\n", err)
				return
			}

			result, err := registry.CallFunction(command)
			if err != nil {
				cmd.PrintErrf("Error calling function %s: %v\n", command.Name, err)
				return
			}

			cmd.Printf("%v\n", result)
		},
	}

	return cmd
}
