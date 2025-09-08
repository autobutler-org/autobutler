package types

import (
	"github.com/nbio/xml"
)

// Numbering Definition Instance Reference
type NumberingProperty struct {
	XMLName xml.Name `xml:"w:numPr"`

	//Numbering Level Reference
	ILvl *DecimalNum `xml:"w:ilvl"`

	//Numbering Definition Instance Reference
	NumID *DecimalNum `xml:"w:numId"`
}

// NewNumberingProperty creates a new NumberingProperty instance.
func NewNumberingProperty() *NumberingProperty {
	return &NumberingProperty{
		ILvl:  NewDecimalNum(0),
		NumID: NewDecimalNum(1),
	}
}

// func (n NumProp) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
// 	start.Name.Local = "w:numPr"

// 	err := e.EncodeToken(start)
// 	if err != nil {
// 		return err
// 	}

// 	if n.ILvl != nil {
// 		if err = n.ILvl.MarshalXML(e, xml.StartElement{
// 			Name: xml.Name{Local: "w:ilvl"},
// 		}); err != nil {
// 			return fmt.Errorf("ILvl: %w", err)
// 		}
// 	}

// 	if n.NumID != nil {
// 		if err = n.NumID.MarshalXML(e, xml.StartElement{
// 			Name: xml.Name{Local: "w:numId"},
// 		}); err != nil {
// 			return fmt.Errorf("NumID: %w", err)
// 		}
// 	}

// 	return e.EncodeToken(start.End())
// }
