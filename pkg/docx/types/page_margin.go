package types

import (
	"github.com/nbio/xml"
)

// PageMargin represents the page margins of a Word document.
type PageMargin struct {
	XMLName xml.Name `xml:"w:pgMar"`

	Left   int `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:left,attr,omitempty"`
	Right  int `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:right,attr,omitempty"`
	Gutter int `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:gutter,attr,omitempty"`
	Header int `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:header,attr,omitempty"`
	Top    int `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:top,attr,omitempty"`
	Footer int `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:footer,attr,omitempty"`
	Bottom int `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:bottom,attr,omitempty"`
}

func NewPageMargin() *PageMargin {
	return &PageMargin{
		Left:   1440,
		Right:  1440,
		Gutter: 0,
		Header: 720,
		Top:    1440,
		Footer: 720,
		Bottom: 1440,
	}
}
