package types

type Paragraph struct {
	id string

	// Attributes
	RsidRPr      *Hex // Revision Identifier for Paragraph Glyph Formatting
	RsidR        *Hex // Revision Identifier for Paragraph
	RsidDel      *Hex // Revision Identifier for Paragraph Deletion
	RsidP        *Hex // Revision Identifier for Paragraph Properties
	RsidRDefault *Hex // Default Revision Identifier for Runs

	// 1. Paragraph Properties
	Property *ParagraphProperty
}
