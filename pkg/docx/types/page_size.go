package types

import (
	"github.com/nbio/xml"
)

// Page Size : w:pgSz
type PageSize struct {
	XMLName xml.Name `xml:"w:pgSz"`

	Width  uint64     `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:w,attr,omitempty"`
	Height uint64     `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:h,attr,omitempty"`
	Orient PageOrient `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:orient,attr,omitempty"`
	Code   int        `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:code,attr,omitempty"`
}

func NewPageSize() *PageSize {
	return &PageSize{
		Width:  12240,
		Height: 15840,
		Orient: PageOrientPortrait,
	}
}
