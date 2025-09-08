package docx

import (
	"autobutler/pkg/docx/types"

	"github.com/nbio/xml"
)

// This element specifies the contents of the body of the document – the main document editing surface.
type Body struct {
	XMLName xml.Name `xml:"w:body"`

	Root            *RootDoc `xml:"-"`
	Children        []BodyChild
	SectionProperty *types.SectionProperty
}

// BodyChild represents a child element within a Word document, which can be a Paragraph.
type BodyChild interface {
	BodyChild()
}

// Use this function to initialize a new Body before adding content to it.
func NewBody(root *RootDoc) *Body {
	return &Body{
		Root:            root,
		SectionProperty: types.NewSectionProperty(),
	}
}

// UnmarshalXML implements xml.Unmarshaler for Body.
// Necessary to handle unmarshalling BodyChild interface types.
func (b *Body) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	b.XMLName = start.Name
	b.Children = []BodyChild{}
	b.SectionProperty = types.NewSectionProperty()

	for {
		tok, err := d.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}

		switch el := tok.(type) {
		case xml.StartElement:
			switch el.Name.Local {
			case "p": // Paragraph
				var p Paragraph
				if err := d.DecodeElement(&p, &el); err != nil {
					return err
				}
				p.Root = b.Root
				b.Children = append(b.Children, &p)
			case "sectPr": // SectionProperty
				var sectPr types.SectionProperty
				if err := d.DecodeElement(&sectPr, &el); err != nil {
					return err
				}
				b.SectionProperty = &sectPr
			// Add more cases here for other BodyChild types if needed
			default:
				// Skip unknown elements
				if err := d.Skip(); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if el.Name == start.Name {
				return nil
			}
		}
	}
	return nil
}
