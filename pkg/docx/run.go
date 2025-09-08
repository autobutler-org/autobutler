package docx

import (
	"autobutler/pkg/docx/types"
	"github.com/nbio/xml"
)

type Run struct {
	XMLName xml.Name `xml:"w:r"`

	Root    *RootDoc   `xml:"-"` // root is the root document to which this run belongs.
	RsidRPr *types.Hex `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:rsidRPr,attr"`
	RsidR   *types.Hex `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:rsidR,attr"`   // Revision Identifier for Paragraph
	RsidDel *types.Hex `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:rsidDel,attr"` // Revision Identifier for Paragraph Deletion

	// Sequence:

	//1. Run Properties
	Property *types.RunProperty

	// 2. Choice - Run Inner content
	Children []RunChild
}

type RunChild interface {
	RunChild()
}

func (r Run) ParagraphChild() {
	// Mark struct for marshalling
}

func NewRun(root *RootDoc) *Run {
	return &Run{
		Root:     root,
		RsidRPr:  types.NewHex(0),
		RsidR:    types.NewHex(0),
		RsidDel:  types.NewHex(0),
		Property: types.NewRunProperty(),
	}
}

// UnmarshalXML implements xml.Unmarshaler for Run.
// Necessary to handle unmarshalling RunChild interface types.
func (r *Run) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	r.XMLName = start.Name
	r.Children = []RunChild{}
	r.Property = types.NewRunProperty()

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
				case "rsidRPr":
					r.RsidRPr = types.NewHexFromString(attr.Value)
				case "rsidR":
					r.RsidR = types.NewHexFromString(attr.Value)
				case "rsidDel":
					r.RsidDel = types.NewHexFromString(attr.Value)
				}
			}
			// Clear attributes so we don't parse them again
			start.Attr = nil
		}

		switch el := tok.(type) {
		case xml.StartElement:
			switch el.Name.Local {
			case "rPr":
				var rPr types.RunProperty
				if err := d.DecodeElement(&rPr, &el); err != nil {
					return err
				}
				r.Property = &rPr
			case "t":
				var t types.Text
				if err := d.DecodeElement(&t, &el); err != nil {
					return err
				}
				r.Children = append(r.Children, &t)
			// Add more cases here for other RunChild types if needed
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

// getProp returns the run properties. If not initialized, it creates and returns a new instance.
func (r *Run) getProp() *types.RunProperty {
	if r.Property == nil {
		r.Property = &types.RunProperty{}
	}
	return r.Property
}

// Sets the color of the Run.
//
// Example:
//
//	modifiedRun := run.Color("FF0000")
//
// Parameters:
//   - colorCode: A string representing the color code (e.g., "FF0000" for red).
//
// Returns:
//   - *Run: The modified Run instance with the updated color.
func (r *Run) Color(colorCode string) *Run {
	r.getProp().Color = types.NewColor(colorCode)
	return r
}

// Sets the size of the Run.

// This method takes an integer parameter representing the desired font size.
// It updates the size property of the Run instance with the specified size,
// Example:

// 	modifiedRun := run.Size(12)

// Parameters:
//   - size: An integer representing the font size.

// Returns:
//   - *Run: The modified Run instance with the updated size.
func (r *Run) Size(size uint64) *Run {
	r.getProp().Size = types.NewFontSize(size * 2)
	return r
}

// Font sets the font for the run.
func (r *Run) Font(font string) *Run {
	if r.getProp().Fonts == nil {
		r.getProp().Fonts = &types.RunFonts{}
	}

	r.getProp().Fonts.Ascii = font
	r.getProp().Fonts.HAnsi = font
	return r
}

// AddHighlight sets the highlight color for the run.
func (r *Run) Highlight(color string) *Run {
	r.getProp().Highlight = types.NewCTString(color)
	return r
}

// AddBold enables bold formatting for the run.
func (r *Run) Bold(value bool) *Run {
	r.getProp().Bold = types.OnOffFromBool(value)
	return r
}

// Italic enables or disables italic formatting for the run.
func (r *Run) Italic(value bool) *Run {
	r.getProp().Italic = types.OnOffFromBool(value)
	return r
}

// Specifies that the contents of this run shall be displayed with a single horizontal line through the center of the line.
func (r *Run) Strike(value bool) *Run {
	r.getProp().Strike = types.OnOffFromBool(value)
	return r
}

// Specifies that the contents of this run shall be displayed with two horizontal lines through each character displayed on the line
func (r *Run) DoubleStrike(value bool) *Run {
	r.getProp().DoubleStrike = types.OnOffFromBool(value)
	return r
}

// Display All Characters As Capital Letters
// Any lowercase characters in this text run shall be formatted for display only as their capital letter character equivalents
func (r *Run) Caps(value bool) *Run {
	r.getProp().Caps = types.OnOffFromBool(value)
	return r
}

// Specifies that all small letter characters in this text run shall be formatted for display only as their capital letter character equivalents
func (r *Run) SmallCaps(value bool) *Run {
	r.getProp().Caps = types.OnOffFromBool(value)
	return r
}

// Outline enables or disables outline formatting for the run.
func (r *Run) Outline(value bool) *Run {
	r.getProp().Outline = types.OnOffFromBool(value)
	return r
}

// Shadow enables or disables shadow formatting for the run.
func (r *Run) Shadow(value bool) *Run {
	r.getProp().Shadow = types.OnOffFromBool(value)
	return r
}

// Emboss enables or disables embossing formatting for the run.
func (r *Run) Emboss(value bool) *Run {
	r.getProp().Emboss = types.OnOffFromBool(value)
	return r
}

// Imprint enables or disables imprint formatting for the run.
func (r *Run) Imprint(value bool) *Run {
	r.getProp().Imprint = types.OnOffFromBool(value)
	return r
}

// Do Not Check Spelling or Grammar
func (r *Run) NoGrammer(value bool) *Run {
	r.getProp().NoGrammar = types.OnOffFromBool(value)
	return r
}

// Use Document Grid Settings For Inter-Character Spacing
func (r *Run) SnapToGrid(value bool) *Run {
	r.getProp().SnapToGrid = types.OnOffFromBool(value)
	return r
}

// Hidden Text
func (r *Run) HideText(value bool) *Run {
	r.getProp().Vanish = types.OnOffFromBool(value)
	return r
}

// Spacing sets the spacing between characters in the run.
func (r *Run) Spacing(value int) *Run {
	r.getProp().Spacing = types.NewDecimalNum(value)
	return r
}

// Underline sets the underline style for the run.
func (r *Run) Underline(value types.Underline) *Run {
	r.getProp().Underline = types.NewGenSingleStrVal(value)
	return r
}

// Style sets the style of the run.
func (r *Run) Style(value string) *Run {
	r.getProp().Style = types.NewRunStyle(value)
	return r
}
