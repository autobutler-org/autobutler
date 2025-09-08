package docx

import (
	"errors"

	"github.com/nbio/xml"

	"autobutler/pkg/docx/constants"
	"autobutler/pkg/docx/types"
)

// Paragraph represents a paragraph in a DOCX document.
type Paragraph struct {
	XMLName xml.Name `xml:"w:p"`

	Root *RootDoc   `xml:"-"` // root is a reference to the root document.
	Id   *types.Hex `xml:"http://schemas.microsoft.com/office/word/2010/wordml w14:paraId,attr"`

	// Attributes
	RsidRPr      *types.Hex `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:rsidRPr,attr"`      // Revision Identifier for Paragraph Glyph Formatting
	RsidR        *types.Hex `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:rsidR,attr"`        // Revision Identifier for Paragraph
	RsidDel      *types.Hex `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:rsidDel,attr"`      // Revision Identifier for Paragraph Deletion
	RsidP        *types.Hex `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:rsidP,attr"`        // Revision Identifier for Paragraph Properties
	RsidRDefault *types.Hex `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:rsidRDefault,attr"` // Default Revision Identifier for Runs

	// 1. Paragraph Properties
	Property *types.ParagraphProperty

	// 2. Choices (Slice of Child elements)
	Children []ParagraphChild
}

func (p Paragraph) BodyChild() {
	// Mark struct for marshalling
}

type ParagraphChild interface {
	ParagraphChild()
}

// UnmarshalXML implements xml.Unmarshaler for Paragraph.
// Necessary to handle unmarshalling ParagraphChild interface types.
func (p *Paragraph) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	p.XMLName = start.Name
	p.Children = []ParagraphChild{}
	p.Property = types.NewParagraphProperty()

	for {
		tok, err := d.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}

		// Parse attributes from the start element
		if start.Attr != nil {
			for _, attr := range start.Attr {
				switch attr.Name.Local {
				case "paraId":
					p.Id = types.NewHexFromString(attr.Value)
				case "rsidRPr":
					p.RsidRPr = types.NewHexFromString(attr.Value)
				case "rsidR":
					p.RsidR = types.NewHexFromString(attr.Value)
				case "rsidDel":
					p.RsidDel = types.NewHexFromString(attr.Value)
				case "rsidP":
					p.RsidP = types.NewHexFromString(attr.Value)
				case "rsidRDefault":
					p.RsidRDefault = types.NewHexFromString(attr.Value)
				}
			}
			// Clear attributes so we don't parse them again
			start.Attr = nil
		}

		switch el := tok.(type) {
		case xml.StartElement:
			switch el.Name.Local {
			case "pPr":
				var pPr types.ParagraphProperty
				if err := d.DecodeElement(&pPr, &el); err != nil {
					return err
				}
				p.Property = &pPr
			case "r":
				var r Run
				if err := d.DecodeElement(&r, &el); err != nil {
					return err
				}
				r.Root = p.Root
				p.Children = append(p.Children, &r)
			case "hyperlink":
				var h Hyperlink
				if err := d.DecodeElement(&h, &el); err != nil {
					return err
				}
				h.Root = p.Root
				p.Children = append(p.Children, &h)
			// Add more cases here for other ParagraphChild types if needed
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

// paraOption defines a type for functions that can configure a Paragraph.
type paraOption func(*Paragraph)

var paragraphId uint64 = 0

// NewParagraph creates and initializes a new Paragraph instance with given options.
func NewParagraph(root *RootDoc) *Paragraph {
	paragraphId++
	id := types.NewHex(paragraphId)
	p := &Paragraph{
		Root:         root,
		Id:           id,
		RsidRPr:      types.NewHex(0),
		RsidR:        types.NewHex(0),
		RsidDel:      types.NewHex(0),
		RsidP:        types.NewHex(0),
		RsidRDefault: types.NewHex(0),
		Property:     types.NewParagraphProperty(),
	}
	return p
}

// AddParagraph adds a new paragraph with the specified text to the document.
// It returns the created Paragraph instance.
//
// Parameters:
//   - text: The text to be added to the paragraph.
//
// Returns:
//   - p: The created Paragraph instance.
func (rd *RootDoc) AddParagraph(text string) *Paragraph {
	p := NewParagraph(rd)
	p.AddText(text)
	rd.Document.Body.Children = append(rd.Document.Body.Children, p)

	return p
}

// Style sets the paragraph style.
//
// Parameters:
//   - value: A string representing the style value. It can be any valid style defined in the WordprocessingML specification.
//
// Example:
//
//	p1 := document.AddParagraph("Example para")
//	paragraph.Style("List Number")
func (p *Paragraph) Style(value string) {
	p.Property.Style = types.NewParagraphStyle(value)
}

// Justification sets the paragraph justification type.
//
// Parameters:
//   - value: A value of type types.Justification representing the justification type.
//     It can be one of the Justification type values defined in the types package.
//
// Example:
//
//	p1 := document.AddParagraph("Example justified para")
//	p1.Justification(types.JustificationCenter) // Center justification
func (p *Paragraph) Justification(value types.Justification) {
	p.Property.Justification = types.NewGenSingleStrVal(value)
}

// Numbering sets the paragraph numbering properties.
//
// This function assigns a numbering definition ID and a level to the paragraph,
// which affects how numbering is displayed in the document.
//
// Parameters:
//   - id: An integer representing the numbering definition ID.
//   - level: An integer representing the level within the numbering definition.
//
// Example:
//
//	p1 := document.AddParagraph("Example numbered para")
//	p1.Numbering(1, 0)
//
// In this example, the paragraph p1 is assigned the numbering properties
// defined by numbering definition ID 1 and level 0.
func (p *Paragraph) Numbering(id int, level int) {
	if p.Property.NumProp == nil {
		p.Property.NumProp = &types.NumberingProperty{}
	}

	p.Property.NumProp.NumID = types.NewDecimalNum(id)
	p.Property.NumProp.ILvl = types.NewDecimalNum(level)
}

// Indent sets the paragraph indentation properties.
//
// This function assigns an indent definition to the paragraph,
// which affects how exactly the paragraph is going to be indented.
//
// Parameters:
//   - indentProp: A types.Indent instance pointer representing exact indentation
//     measurements to use.
//
// Example:
//
//	var size360 int = 360
//	var sizeu420 uint64 = 420
//	indent360 := types.Indent{Left: &size360, Hanging: &sizeu420}
//
//	p1 := document.AddParagraph("Example indented para")
//	p1.Indent(&indent360)
func (p *Paragraph) Indent(indentProp *types.Indent) {
	p.Property.Indent = indentProp
}

// Appends a new text to the Paragraph.
// Example:
//
//	paragraph := AddParagraph()
//	modifiedParagraph := paragraph.AddText("Hello, World!")
//
// Parameters:
//   - text: A string representing the text to be added to the Paragraph.
//
// Returns:
//   - *Run: The newly created Run instance added to the Paragraph.
func (p *Paragraph) AddText(text string) *Run {
	t := types.NewText(text)

	runChildren := []RunChild{}
	runChildren = append(runChildren, t)
	run := NewRun(p.Root)
	run.Children = runChildren
	p.Children = append(p.Children, run)
	return run
}

// AddEmptyParagraph adds a new empty paragraph to the document.
// It returns the created Paragraph instance.
//
// Returns:
//   - p: The created Paragraph instance.
func (rd *RootDoc) AddEmptyParagraph() *Paragraph {
	p := NewParagraph(rd)

	rd.Document.Body.Children = append(rd.Document.Body.Children, p)

	return p
}

func AddParagraph(root *RootDoc, text string) *Paragraph {
	p := NewParagraph(root)
	p.AddText(text)
	return p
}

func (p *Paragraph) AddRun() *Run {
	run := NewRun(p.Root)
	p.Children = append(p.Children, run)
	return run
}

// GetStyle retrieves the style information applied to the Paragraph.
//
// Returns:
//   - *types.Style: The style information of the Paragraph.
//   - error: An error if the style information is not found.
func (p *Paragraph) GetStyle() (*types.Style, error) {
	if p.Property == nil || p.Property.Style == nil {
		return nil, errors.New("No property for the style")
	}

	style := p.Root.GetStyleByID(p.Property.Style.Val, types.StyleTypeParagraph)
	if style == nil {
		return nil, errors.New("No style found for the paragraph")
	}

	return style, nil
}

func (p *Paragraph) AddLink(text string, link string) *Hyperlink {
	rId := p.Root.Document.addLinkRelation(link)

	runChildren := []RunChild{}
	runChildren = append(runChildren, types.NewText(text))
	run := NewRun(p.Root)
	run.Children = runChildren
	run.Property = &types.RunProperty{
		Style: &types.CTString{
			Val: constants.HyperLinkStyle,
		},
	}

	hyperLink := &Hyperlink{
		Root: p.Root,
		Id:   rId,
		Run:  run,
	}

	p.Children = append(p.Children, hyperLink)

	return hyperLink
}
