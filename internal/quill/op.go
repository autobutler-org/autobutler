package quill

type Op struct {
	Insert     string                 `json:"insert"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}
