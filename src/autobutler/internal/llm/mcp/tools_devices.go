package mcp

const (
	LightStateOn  string = "on"
	LightStateOff string = "off"
)

type LightParams struct {
	Location   string `json:"param0"`
	LightState string `json:"param1"`
}

func (p LightParams) Output(result any) (string, []any) {
	response := result.(LightResponse)
	return "Turned %s light %s", []any{p.Location, response.State}
}

type LightResponse struct {
	State string `json:"state"`
}

func (r McpRegistry) SetLightState(location string, state string) LightResponse {
	return LightResponse{
		State: state,
	}
}
