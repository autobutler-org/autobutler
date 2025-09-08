package types

// PageNumberingType represents the page numbering format in a Word document.
type PageNumberingType struct {
	Format NumFmt `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:fmt,attr,omitempty"`
	Start  int    `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:start,attr,omitempty"`
}

func NewPageNumberingType() *PageNumberingType {
	return &PageNumberingType{
		Start: 1,
	}
}
