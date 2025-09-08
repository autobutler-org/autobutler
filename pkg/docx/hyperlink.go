package docx

import (
	"autobutler/pkg/docx/types"
	"github.com/nbio/xml"
)

type Hyperlink struct {
	XMLName xml.Name `xml:"w:hyperlink"`

	Root *RootDoc `xml:"-"` // root is the root document to which this hyperlink belongs.
	Id   string   `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
	Run  *Run
}

func (h Hyperlink) ParagraphChild() {
	// Mark struct for marshalling
}

// getProp returns the hyperlink properties. If not initialized, it creates and returns a new instance.
func (h *Hyperlink) getProp() *types.RunProperty {
	if h.Run.Property == nil {
		h.Run.Property = types.NewRunProperty()
	}
	return h.Run.Property
}

func (r *Hyperlink) Color(colorCode string) *Hyperlink {
	r.getProp().Color = types.NewColor(colorCode)
	return r
}

func (r *Hyperlink) Size(size uint64) *Hyperlink {
	r.getProp().Size = types.NewFontSize(size * 2)
	return r
}

// Font sets the font for the hyperlink.
func (r *Hyperlink) Font(font string) *Hyperlink {
	if r.getProp().Fonts == nil {
		r.getProp().Fonts = &types.RunFonts{}
	}

	r.getProp().Fonts.Ascii = font
	r.getProp().Fonts.HAnsi = font
	return r
}

// AddBold enables bold formatting for the hyperlink.
func (r *Hyperlink) Bold(value bool) *Hyperlink {
	r.getProp().Bold = types.OnOffFromBool(value)
	return r
}

// Italic enables or disables italic formatting for the hyperlink.
func (r *Hyperlink) Italic(value bool) *Hyperlink {
	r.getProp().Italic = types.OnOffFromBool(value)
	return r
}

// Specifies that the contents of this hyperlink shall be displayed with a single horizontal line through the center of the line.
func (r *Hyperlink) Strike(value bool) *Hyperlink {
	r.getProp().Strike = types.OnOffFromBool(value)
	return r
}

// Specifies that the contents of this hyperlink shall be displayed with two horizontal lines through each character displayed on the line
func (r *Hyperlink) DoubleStrike(value bool) *Hyperlink {
	r.getProp().DoubleStrike = types.OnOffFromBool(value)
	return r
}

// Display All Characters As Capital Letters
// Any lowercase characters in this text hyperlink shall be formatted for display only as their capital letter character equivalents
func (r *Hyperlink) Caps(value bool) *Hyperlink {
	r.getProp().Caps = types.OnOffFromBool(value)
	return r
}

// Underline sets the underline style for the hyperlink.
func (r *Hyperlink) Underline(value types.Underline) *Hyperlink {
	r.getProp().Underline = types.NewGenSingleStrVal(value)
	return r
}

// Style sets the style of the Hyperlink.
func (r *Hyperlink) Style(value string) *Hyperlink {
	r.getProp().Style = types.NewRunStyle(value)
	return r
}
