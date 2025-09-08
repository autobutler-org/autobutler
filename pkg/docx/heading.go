package docx

import (
	"autobutler/pkg/docx/types"
	"errors"
	"fmt"
)

// Return a heading paragraph newly added to the end of the document.
// The heading paragraph will contain text and have its paragraph style determined by level.
// If level is 0, the style is set to Title.
// The style is set to Heading {level}.
// if level is outside the range 0-9, error will be returned
func (rd *RootDoc) AddHeading(text string, level uint) (*Paragraph, *Run, error) {
	if level > 9 {
		return nil, nil, errors.New("Heading level not supported")
	}

	p := NewParagraph(rd)
	p.Property = types.NewParagraphProperty()

	style := "Title"
	if level != 0 {
		style = fmt.Sprintf("Heading%d", level)
	}

	p.Property.Style = types.NewParagraphStyle(style)

	rd.Document.Body.Children = append(rd.Document.Body.Children, p)

	run := p.AddText(text)
	return p, run, nil
}

func (rd *RootDoc) AddHeadingHyperlink(text string, link string, level uint) (*Paragraph, *Hyperlink, error) {
	if level > 9 {
		return nil, nil, errors.New("Heading level not supported")
	}

	p := NewParagraph(rd)
	p.Property = types.NewParagraphProperty()

	style := "Title"
	if level != 0 {
		style = fmt.Sprintf("Heading%d", level)
	}

	p.Property.Style = types.NewParagraphStyle(style)

	rd.Document.Body.Children = append(rd.Document.Body.Children, p)

	h := p.AddLink(text, link)
	return p, h, nil
}
