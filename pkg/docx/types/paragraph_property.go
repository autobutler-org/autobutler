package types

import (
	"github.com/nbio/xml"
)

// Numbering Level Associated Paragraph Properties
// TODO: 194_doc-support Implement all the marshalling here
type ParagraphProperty struct {
	XMLName xml.Name `xml:"w:pPr"`

	// 1. This element specifies the style ID of the paragraph style which shall be used to format the contents of this paragraph.
	Style *CTString `xml:"pStyle,omitempty"`

	// 2. Keep Paragraph With Next Paragraph
	KeepNext *OnOff `xml:"keepNext,omitempty"`

	// 3. Keep All Lines On One Page
	KeepLines *OnOff `xml:"keepLines,omitempty"`

	// 4. Start Paragraph on Next Page
	PageBreakBefore *OnOff `xml:"pageBreakBefore,omitempty"`

	// 6. Allow First/Last Line to Display on a Separate Page
	WindowControl *OnOff `xml:"widowControl,omitempty"`

	// 7. Numbering Definition Instance Reference
	NumProp *NumberingProperty

	// 8. Suppress Line Numbers for Paragraph
	SuppressLineNmbrs *OnOff `xml:"suppressLineNumbers,omitempty"`

	// 12. Suppress Hyphenation for Paragraph
	SuppressAutoHyphens *OnOff `xml:"suppressAutoHyphens,omitempty"`

	// 13. Use East Asian Typography Rules for First and Last Character per Line
	Kinsoku *OnOff `xml:"kinsoku,omitempty"`

	// 14. Allow Line Breaking At Character Level
	WordWrap *OnOff `xml:"wordWrap,omitempty"`

	// 15. Allow Punctuation to Extent Past Text Extents
	OverflowPunct *OnOff `xml:"overflowPunct,omitempty"`

	// 16. Compress Punctuation at Start of a Line
	TopLinePunct *OnOff `xml:"topLinePunct,omitempty"`

	// 17. Automatically Adjust Spacing of Latin and East Asian Text
	AutoSpaceDE *OnOff `xml:"autoSpaceDE,omitempty"`

	// 18. Automatically Adjust Spacing of East Asian Text and Numbers
	AutoSpaceDN *OnOff `xml:"autoSpaceDN,omitempty"`

	// 19. Right to Left Paragraph Layout
	Bidi *OnOff `xml:"bidi,omitempty"`

	// 20. Automatically Adjust Right Indent When Using Document Grid
	AdjustRightInd *OnOff `xml:"adjustRightInd,omitempty"`

	// 21. Use Document Grid Settings for Inter-Line Paragraph Spacing
	SnapToGrid *OnOff `xml:"snapToGrid,omitempty"`

	// 23. Paragraph Indentation
	Indent *Indent `xml:"ind,omitempty"`

	// 24. Ignore Spacing Above and Below When Using Identical Styles
	CtxlSpacing *OnOff `xml:"contextualSpacing,omitempty"`

	// 25. Use Left/Right Indents as Inside/Outside Indents
	MirrorIndents *OnOff `xml:"mirrorIndents,omitempty"`

	// 26. Prevent Text Frames From Overlapping
	SuppressOverlap *OnOff `xml:"suppressOverlap,omitempty"`

	// 27. Paragraph Alignment
	Justification *GenSingleStrVal[Justification] `xml:"jc,omitempty"`

	// 31. Associated Outline Level
	OutlineLvl *DecimalNum `xml:"outlineLvl,omitempty"`

	// 32. Associated HTML div ID
	DivID *DecimalNum `xml:"divId,omitempty"`

	// 33. Paragraph Conditional Formatting
	CnfStyle *CTString `xml:"cnfStyle,omitempty"`

	// 34. Run Properties for the Paragraph Mark
	RunProperty *RunProperty

	// 35. Section Properties
	SectPr *SectionProperty `xml:"sectPr,omitempty"`

	// 36. Revision Information for Paragraph Properties
	PPrChange *PPrChange `xml:"pPrChange,omitempty"`
}

func NewParagraphProperty() *ParagraphProperty {
	// TODO: 194_doc-support Fill the defaults here
	/*
	   <w:pPr>
	   	<w:numPr>
	   		<w:ilvl w:val="0" />
	   		<w:numId w:val="1" />
	   	</w:numPr>
	   	<w:spacing w:after="0" w:afterAutospacing="0" />
	   	<w:ind w:left="720" w:hanging="360" />
	   	<w:rPr>
	   		<w:u w:val="none" />
	   	</w:rPr>
	   </w:pPr>
	*/
	return &ParagraphProperty{
		NumProp: NewNumberingProperty(),
		// TODO: 194_doc-support implement a spacing type at some point
		// Spacing: NewSpacing(0, 0),
		Indent:      NewIndent(),
		RunProperty: NewRunProperty(),
	}
}

type binElems struct {
	elem    *OnOff
	XMLName string
}

// NewParagraphStyle creates a new ParagraphStyle.
func NewParagraphStyle(val string) *CTString {
	return &CTString{Val: val}
}

// DefaultParagraphStyle creates the default ParagraphStyle with the value "Normal".
func DefaultParagraphStyle() *CTString {
	return &CTString{Val: "Normal"}
}

// <== ParaProp ends here ==>

// Revision Information for Paragraph Properties
type PPrChange struct {
	ID       int                `xml:"id,attr"`
	Author   string             `xml:"author,attr"`
	Date     *string            `xml:"date,attr,omitempty"`
	ParaProp *ParagraphProperty `xml:"pPr"`
}

// <== PPrChange ends here ==>
