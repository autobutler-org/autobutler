package quill

type Delta struct {
	Ops []Op `json:"ops"`
}

var ExampleDelta = Delta{
	Ops: []Op{
		{Insert: "Gandalf", Attributes: map[string]interface{}{"bold": true}},
		{Insert: " the "},
		{Insert: "Grey", Attributes: map[string]interface{}{"color": "#cccccc"}},
	},
}
