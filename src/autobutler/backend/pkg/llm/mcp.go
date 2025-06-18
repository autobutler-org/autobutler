package llm

import (
	"errors"
)

// Function describes a callable functionality with its name and parameter schema.
type Function struct {
	Name       string
	Parameters []string // List of parameter names
	Handler    func(params map[string]interface{}) (interface{}, error)
}

type MCPCommand struct {
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters"` // Parameters as a map
}

// MCPRegistry manages registered functionalities.
type MCPRegistry struct {
	functions map[string]*Function
}

// NewMCPRegistry creates a new MCPRegistry.
func NewMCPRegistry() *MCPRegistry {
	return &MCPRegistry{
		functions: make(map[string]*Function),
	}
}

// RegisterFunction registers a new functionality.
func (r *MCPRegistry) RegisterFunction(f *Function) {
	r.functions[f.Name] = f
}

// ListFunctions returns a list of registered functionalities and their parameters.
func (r *MCPRegistry) ListFunctions() []map[string]interface{} {
	list := []map[string]interface{}{}
	for _, f := range r.functions {
		list = append(list, map[string]interface{}{
			"name":       f.Name,
			"parameters": f.Parameters,
		})
	}
	return list
}

// CallFunction invokes a registered function by name with JSON parameters.
func (r *MCPRegistry) CallFunction(command MCPCommand) (interface{}, error) {
	f, ok := r.functions[command.Name]
	if !ok {
		return nil, errors.New("function not found")
	}
	return f.Handler(command.Parameters)
}

// Example usage:

// func main() {
// 	reg := NewMCPRegistry()
// 	reg.RegisterFunction(&Function{
// 		Name:       "add",
// 		Parameters: []string{"a", "b"},
// 		Handler: func(params map[string]interface{}) (interface{}, error) {
// 			a, okA := params["a"].(float64)
// 			b, okB := params["b"].(float64)
// 			if !okA || !okB {
// 				return nil, errors.New("invalid parameters")
// 			}
// 			return a + b, nil
// 		},
// 	})
// 	fmt.Println(reg.ListFunctions())
// 	result, err := reg.CallFunction("add", []byte(`{"a":2,"b":3}`))
// 	fmt.Println(result, err) // Output: 5 <nil>
// }
