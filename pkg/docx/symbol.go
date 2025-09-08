package docx

// Symbol represents a symbol character in a document.
type Symbol struct {
	Font *string `xml:"font,attr,omitempty"`
	Char *string `xml:"char,attr,omitempty"`
}

func NewSymbol(font, char string) *Symbol {
	return &Symbol{
		Font: &font,
		Char: &char,
	}
}
