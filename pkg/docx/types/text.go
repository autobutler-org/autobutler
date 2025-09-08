package types

import (
	"github.com/nbio/xml"
)

type Text struct {
	XMLName xml.Name `xml:"w:t"`

	Text  string    `xml:",chardata"`
	Space TextSpace `xml:"http://www.w3.org/XML/1998/namespace xml:space,attr"`
}

func (t Text) RunChild() {
	// Mark struct for marshalling
}
