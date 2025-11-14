package quill

type Delta struct {
	Ops []Op `json:"ops"`
}

var ExampleDelta = Delta{
	Ops: []Op{
		{Insert: "Gandalf", Attributes: map[string]any{"bold": true}},
		{Insert: " the "},
		{Insert: "Grey", Attributes: map[string]any{"color": "#cccccc"}},
	},
}
