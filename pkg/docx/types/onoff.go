package types

import (
	"errors"

	"github.com/nbio/xml"
)

// Optional Bool Element: Helper element that only has one attribute which is optional
type OnOff struct {
	Val *OnOffValue `xml:"val,attr,omitempty"`
}

func OnOffFromBool(value bool) *OnOff {
	o := OnOffFalse
	if value {
		o = OnOffTrue
	}

	return &OnOff{
		Val: &o,
	}
}

func OnOffFromStr(value string) (*OnOff, error) {
	o, err := OnOffValueFromStr(value)
	if err != nil {
		return nil, err
	}
	return &OnOff{
		Val: &o,
	}, nil
}

// Disable sets the value to false and valexists true
func (n *OnOff) Disable() {
	o := OnOffFalse
	n.Val = &o
}

// MarshalXML implements the xml.Marshaler interface for the Bold type.
// It encodes the instance into XML using the "w:XMLName" element with a "w:val" attribute.
func (n OnOff) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if n.Val != nil { // Add val attribute only if the val exists
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:val"}, Value: string(*n.Val)})
	}

	err := e.EncodeToken(start)
	if err != nil {
		return err
	}
	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

// This simple type specifies a set of values for any binary (on or off) property defined in a WordprocessingML document.
// A value of on, 1, or true specifies that the property shall be turned on. This is the default value for this attribute, and is implied when the parent element is present, but this attribute is omitted.
//
// A value of off, 0, or false specifies that the property shall be explicitly turned off.
type OnOffValue string

const (
	OnOffZero  OnOffValue = "0"
	OnOffOne   OnOffValue = "1"
	OnOffFalse OnOffValue = "false"
	OnOffTrue  OnOffValue = "true"
	OnOffOff   OnOffValue = "off"
	OnOffOn    OnOffValue = "on"
)

func OnOffValueFromStr(s string) (OnOffValue, error) {
	switch s {
	case "0":
		return OnOffZero, nil
	case "1":
		return OnOffOne, nil
	case "false":
		return OnOffFalse, nil
	case "true":
		return OnOffTrue, nil
	case "off":
		return OnOffOff, nil
	case "on":
		return OnOffOn, nil
	default:
		return "", errors.New("invalid OnOff string")
	}
}

func (d *OnOffValue) UnmarshalXMLAttr(attr xml.Attr) error {
	val, err := OnOffValueFromStr(attr.Value)
	if err != nil {
		return err
	}

	*d = val

	return nil

}
