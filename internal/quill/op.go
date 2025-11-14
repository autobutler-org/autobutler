package quill

type Op struct {
	Insert     string         `json:"insert"`
	Attributes map[string]any `json:"attributes,omitempty"`
}
