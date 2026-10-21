package odputil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
)

// The .qslide an import writes, field for field what quark_slides'
// QslideCodec.encode writes and what pptxutil's import writes: the same keys
// in the same order, the same defaults left out, and two-space indentation,
// so a file read back and saved by the editor does not change.

// importSchemaVersion is the schema an import writes. An Impress style
// sheet does not map onto a .qslide theme without loss, so, as pptxutil's
// import does, an import keeps to version 1: every color literal, every run
// styled as it is drawn, and no theme. A transition is written as version 3
// has it and a table as version 4 has it; the codec's migrations carry both
// through unchanged.
const importSchemaVersion = 1

// The element types an import writes.
const (
	typeText  = "text"
	typeShape = "shape"
	typeImage = "image"
	typeLine  = "line"
	typeGroup = "group"
	typeTable = "table"
)

// outSize is the slide size in slide units.
type outSize struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// outSlide is one slide; Elements is never nil.
type outSlide struct {
	ID         string         `json:"id"`
	Background *outBackground `json:"background,omitempty"`
	Elements   []outElement   `json:"elements"`
	Notes      string         `json:"notes,omitempty"`
	Transition *outTransition `json:"transition,omitempty"`
}

// outTransition is a slide's transition; the direction is left out when
// left and the duration when 500 ms, as the codec leaves its defaults out.
type outTransition struct {
	Kind      string `json:"kind"`
	Direction string `json:"direction,omitempty"`
	Duration  int    `json:"duration,omitempty"`
}

// outBackground is a slide's background color, image, or both.
type outBackground struct {
	Color string `json:"color,omitempty"`
	Image string `json:"image,omitempty"`
}

// outElement is one of outText, outShape, outImage, outLine, outGroup and
// outTable.
type outElement any

// outFrame is an element's box; rotation is left out at 0.
type outFrame struct {
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Rotation float64 `json:"rotation,omitempty"`
}

// outText is a text box. Paragraphs is never nil.
type outText struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Frame      outFrame       `json:"frame"`
	Paragraphs []outParagraph `json:"paragraphs"`
	Anchor     string         `json:"anchor,omitempty"`
	AutoFit    string         `json:"autoFit,omitempty"`
}

// outParagraph is one paragraph. Runs is never nil.
type outParagraph struct {
	Runs        []outRun `json:"runs"`
	Align       string   `json:"align,omitempty"`
	LineSpacing *float64 `json:"lineSpacing,omitempty"`
	List        string   `json:"list,omitempty"`
}

// outRun is a styled span of text.
type outRun struct {
	Text          string   `json:"text"`
	Bold          bool     `json:"bold,omitempty"`
	Italic        bool     `json:"italic,omitempty"`
	Underline     bool     `json:"underline,omitempty"`
	Strikethrough bool     `json:"strikethrough,omitempty"`
	FontSize      *float64 `json:"fontSize,omitempty"`
	FontFamily    *string  `json:"fontFamily,omitempty"`
	Color         string   `json:"color,omitempty"`
}

// outShape is a preset shape.
type outShape struct {
	ID           string     `json:"id"`
	Type         string     `json:"type"`
	Frame        outFrame   `json:"frame"`
	Kind         string     `json:"kind"`
	Fill         string     `json:"fill,omitempty"`
	Stroke       *outStroke `json:"stroke,omitempty"`
	CornerRadius *float64   `json:"cornerRadius,omitempty"`
	Opacity      *float64   `json:"opacity,omitempty"`
}

// outStroke is an outline; dash is left out when solid.
type outStroke struct {
	Color string  `json:"color"`
	Width float64 `json:"width"`
	Dash  string  `json:"dash,omitempty"`
}

// outImage is a picture.
type outImage struct {
	ID      string   `json:"id"`
	Type    string   `json:"type"`
	Frame   outFrame `json:"frame"`
	Source  string   `json:"source"`
	AltText string   `json:"altText,omitempty"`
	Fit     string   `json:"fit,omitempty"`
}

// outLine is a straight line; its stroke is always written.
type outLine struct {
	ID       string    `json:"id"`
	Type     string    `json:"type"`
	Frame    outFrame  `json:"frame"`
	Stroke   outStroke `json:"stroke"`
	Flipped  bool      `json:"flipped,omitempty"`
	StartCap string    `json:"startCap,omitempty"`
	EndCap   string    `json:"endCap,omitempty"`
	Opacity  *float64  `json:"opacity,omitempty"`
}

// outGroup is a group; Children is never nil and in group-local frames.
type outGroup struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Frame    outFrame     `json:"frame"`
	Children []outElement `json:"children"`
}

// outTable is a table, in the order TableElement writes its fields: its
// frame the sum of its columns and rows, and a cell for every row and
// column.
type outTable struct {
	ID         string      `json:"id"`
	Type       string      `json:"type"`
	Frame      outFrame    `json:"frame"`
	Columns    []float64   `json:"columns"`
	Rows       []float64   `json:"rows"`
	HeaderRow  bool        `json:"headerRow,omitempty"`
	BandedRows bool        `json:"bandedRows,omitempty"`
	Cells      [][]outCell `json:"cells"`
}

// outCell is one table cell, in the order SlideTableCell writes its fields;
// a blank cell is {}. A span of 1 is left out.
type outCell struct {
	Paragraphs []outParagraph `json:"paragraphs,omitempty"`
	Fill       string         `json:"fill,omitempty"`
	Borders    *outBorders    `json:"borders,omitempty"`
	Anchor     string         `json:"anchor,omitempty"`
	RowSpan    int            `json:"rowSpan,omitempty"`
	ColSpan    int            `json:"colSpan,omitempty"`
}

// outBorders are a cell's lines; a side without one is left out.
type outBorders struct {
	Top    *outStroke `json:"top,omitempty"`
	Right  *outStroke `json:"right,omitempty"`
	Bottom *outStroke `json:"bottom,omitempty"`
	Left   *outStroke `json:"left,omitempty"`
}

// qslideWriter streams a .qslide: the header, then each slide as it is
// converted, then the close. What it writes is byte for byte what encoding
// the whole presentation with two-space indentation would be.
type qslideWriter struct {
	w      io.Writer
	slides int
}

// header writes everything ahead of the first slide.
func (q *qslideWriter) header(title string, size outSize) error {
	t, err := marshalIndented(title, "  ")
	if err != nil {
		return err
	}
	s, err := marshalIndented(size, "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(q.w, "{\n  \"schemaVersion\": %d,\n  \"title\": %s,\n  \"size\": %s,\n  \"slides\": [",
		importSchemaVersion, t, s)
	return err
}

// slide writes one slide.
func (q *qslideWriter) slide(slide outSlide) error {
	body, err := marshalIndented(slide, "    ")
	if err != nil {
		return err
	}
	sep := ",\n    "
	if q.slides == 0 {
		sep = "\n    "
	}
	q.slides++
	_, err = io.WriteString(q.w, sep+body)
	return err
}

// close ends the slide list and the document.
func (q *qslideWriter) close() error {
	end := "\n  ]\n}\n"
	if q.slides == 0 {
		end = "]\n}\n"
	}
	_, err := io.WriteString(q.w, end)
	return err
}

// marshalIndented is v as indented JSON whose continuation lines carry
// prefix. HTML is not escaped, as dart:convert does not escape it.
func marshalIndented(v any, prefix string) (string, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent(prefix, "  ")
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return strings.TrimSuffix(b.String(), "\n"), nil
}

// round keeps two decimals of a converted length.
func round(v float64) float64 {
	r := math.Round(v*100) / 100
	if r == 0 || math.IsNaN(r) {
		// Never -0, which JSON would write as "-0", nor NaN, which it cannot
		// write at all.
		return 0
	}
	return r
}
