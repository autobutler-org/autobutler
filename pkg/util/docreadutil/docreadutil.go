// Package docreadutil reads word-processing documents into one small block
// model, for the read-only document viewer and for "Open as Quark doc"
// (docs/architecture/read-only-documents.md, Decision 2).
//
// A reader turns a document into a stream of blocks: headings, paragraphs
// and list items made of styled runs, tables whose cells hold blocks of their
// own, images, page breaks, and "omitted" markers for what is not shown. The
// stream goes to one of two writers, chosen by [ConvertParams.Output]:
//
//   - [OutputNDJSON] writes one JSON object per line for the viewer: a meta
//     line, one line per top-level block, and an end line carrying the word,
//     character and paragraph counts, whether a limit truncated the document,
//     and its warnings. The end line is always last, so a stream without one
//     was cut off by an error.
//   - [OutputDelta] writes a .qdoc, a Quill Delta. It flattens what a .qdoc
//     cannot hold — a table becomes one line per row with its cells separated
//     by tabs, and images are left out — and names those losses in
//     [ConvertResult.Warnings].
//
// Neither writer builds the document in memory: a block is written as soon as
// it is read, and only a table is held until its last row. Output stops at
// [Limits.MaxBlocks] blocks or [Limits.MaxOutputBytes] bytes, whichever comes
// first, and is then marked truncated rather than failed.
//
// # DOCX and DOCM
//
// The DOCX reader opens the package through officezip and streams
// word/document.xml a token at a time. styles.xml and numbering.xml are read
// first and held, each bounded by [Limits.MaxHeldBytes] and
// [Limits.MaxDefinitions], because a paragraph names its style and its list
// by id. Headings come from outline levels and the Heading and Title styles,
// lists from numbering.xml with their markers counted, and tables carry
// gridSpan and vMerge as column and row spans. Tracked changes read as
// accepted: insertions are kept and deletions dropped. Footnotes and
// endnotes follow the body under a "Notes" heading.
//
// Nothing a document contains is executed or fetched. A DOCM's macros are
// never opened, field codes are not evaluated (only their cached result text
// is shown), embedded OLE objects, charts, shapes, text boxes, equations and
// altChunk documents become omitted blocks, and a linked image becomes an
// image block marked external that names no part. A hyperlink survives only
// with an http, https or mailto target. Headers, footers and comments are
// left out with a warning.
//
// # ODT
//
// The ODT reader opens the package through officezip and streams
// content.xml, whose automatic styles come before its body. styles.xml is
// read first and held, bounded as DOCX's parts are; the style names and
// list styles of both parts together are bounded by [Limits.MaxDefinitions].
// Headings come from text:h and the Title and Heading styles, lists from
// text:list and their list styles with their markers counted, and tables
// carry table:number-columns-spanned and table:number-rows-spanned, with a
// repeated row or cell written out as many times as it repeats, up to a
// bound. White space collapses as ODF says it does. Tracked changes read as
// accepted. A note's body is not held: once the body is written,
// content.xml is read a second time for the bodies of the notes it cited,
// which follow under a "Notes" heading.
//
// As with DOCX, nothing is executed or fetched: scripts and event listeners
// are never read, embedded objects, charts, formulas and shapes become
// omitted blocks, a picture linked from outside the package becomes an
// external image block, and a document whose content is encrypted is
// refused. A text box's paragraphs are shown where the box sits, since that
// is how a captioned picture is written. Headers, footers and comments are
// left out with a warning.
//
// # RTF
//
// The RTF reader is a tokenizer of its own over a buffered reader on the
// file, which it reads once from start to end: RTF is a flat stream of
// groups, control words and text, so it needs no random access. Text is
// decoded from the \uN escapes Unicode is written as, skipping their
// fallback, or else from the code page the document or its font names;
// double-byte code pages written without \uN are not decoded, with a
// warning. Headings come from \outlinelevel and the heading styles of the
// style sheet, lists from \listtext and \pntext, and tables from \trowd
// ... \row, with spans worked out from the cells' edges and from \clmrg
// and \clvmrg, and nested tables from \itap and \nestcell. Footnotes are
// held, bounded by [Limits.MaxHeldBytes], and follow the body under a
// "Notes" heading. Groups nest at most 256 deep outside a group being
// skipped.
//
// Pictures are inline data, shown as omitted blocks rather than images in
// this version. They, \bin data, objects and every destination the reader
// does not know are read through and discarded, never held. Objects,
// shapes and equations become omitted blocks; fields show only their
// result; headers, footers and comments are left out with a warning.
package docreadutil

// cspell:ignore DOCM docm clmrg clvmrg nestcell

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/officezip"
)

// Format is a document format a reader understands.
type Format string

const (
	// FormatDOCX is a Word document.
	FormatDOCX Format = "docx"
	// FormatDOCM is a macro-enabled Word document, read as a DOCX whose
	// macros are never opened.
	FormatDOCM Format = "docm"
	// FormatODT is an OpenDocument text document.
	FormatODT Format = "odt"
	// FormatRTF is a Rich Text Format document.
	FormatRTF Format = "rtf"
)

// Output chooses what [Convert] writes.
type Output int

const (
	// OutputNDJSON writes the block stream for the viewer, one JSON object
	// per line.
	OutputNDJSON Output = iota
	// OutputDelta writes a .qdoc: a Quill Delta, {"ops": [...]}.
	OutputDelta
)

// NDJSONContentType is the media type of the [OutputNDJSON] stream.
const NDJSONContentType = "application/x-ndjson"

// StreamVersion is the version the [OutputNDJSON] meta line carries. It goes
// up when the shape of a line changes in a way an older client cannot read.
const StreamVersion = 1

// Default limits, the bounds the read-only document viewer starts from
// (docs/architecture/read-only-documents.md, "Memory and streaming").
const (
	// DefaultMaxBlocks bounds the blocks of one document, nested ones
	// included.
	DefaultMaxBlocks = 50_000
	// DefaultMaxOutputBytes bounds what one conversion writes.
	DefaultMaxOutputBytes = 16 << 20 // 16 MiB
	// DefaultMaxHeldBytes bounds one part read and held whole: styles,
	// numbering, document properties.
	DefaultMaxHeldBytes = 8 << 20 // 8 MiB
	// DefaultMaxDefinitions bounds the style or numbering definitions one
	// held part declares, and the notes one document refers to.
	DefaultMaxDefinitions = 20_000
	// DefaultMaxTableDepth bounds how deeply tables nest; a table deeper
	// than this is flattened to its cells' blocks.
	DefaultMaxTableDepth = 8
)

var (
	// ErrMalformed is a file that is not a well-formed document of its
	// format. It is [officezip.ErrMalformed], so either name matches.
	ErrMalformed = officezip.ErrMalformed
	// ErrTooLarge is a document past one of the limits that refuse it
	// rather than truncate it: the package's, or a held part's. It is
	// [officezip.ErrTooLarge].
	ErrTooLarge = officezip.ErrTooLarge
	// ErrUnsupportedFormat is a [Format] no reader understands.
	ErrUnsupportedFormat = errors.New("docreadutil: unsupported document format")
)

// Limits bounds what one conversion may cost. A zero field takes its
// Default value, so Limits{} is the default set.
type Limits struct {
	// Zip bounds the office package itself; see [officezip.Limits].
	Zip officezip.Limits
	// MaxBlocks bounds the blocks read, nested ones included. Reaching it
	// truncates the document.
	MaxBlocks int
	// MaxOutputBytes bounds what is written, and what is held for a table
	// not yet written. Reaching it truncates the document.
	MaxOutputBytes int64
	// MaxHeldBytes bounds one part read and held whole. Passing it refuses
	// the document with [ErrTooLarge].
	MaxHeldBytes int64
	// MaxDefinitions bounds the definitions of one held part. Passing it
	// refuses the document with [ErrTooLarge].
	MaxDefinitions int
	// MaxTableDepth bounds how deeply tables nest before they are
	// flattened.
	MaxTableDepth int
}

// ConvertParams says which document to read and where to write it.
type ConvertParams struct {
	// Ctx ends the conversion early when it is done; nil never does.
	Ctx context.Context
	// Source is the document, read in place.
	Source io.ReaderAt
	// Size is Source's length in bytes.
	Size int64
	// Format is the document's format; see [FormatForName].
	Format Format
	// Output chooses the writer.
	Output Output
	// Out receives what the writer writes. Nothing is written until the
	// document has opened, so a file that is not a document of its format
	// fails before the first byte.
	Out io.Writer
	// Limits bounds the conversion.
	Limits Limits
}

// ConvertResult reports what a conversion read and wrote.
type ConvertResult struct {
	// Title is the document's title from its properties, if it names one.
	Title string
	// Truncated reports that a limit stopped the document early.
	Truncated bool
	// Blocks counts the blocks read, nested ones included.
	Blocks int
	// Words, Characters and Paragraphs count what was read: words are
	// separated by white space, characters are every character but line
	// breaks, and paragraphs are the text blocks that hold any text.
	Words, Characters, Paragraphs int
	// Warnings names what was left out, in sentences a user can read: what
	// the reader did not show and, for [OutputDelta], what the Delta could
	// not hold.
	Warnings []string
}

// BlockType names what a [Block] is.
type BlockType string

// The block types.
const (
	BlockHeading   BlockType = "heading"
	BlockParagraph BlockType = "paragraph"
	BlockListItem  BlockType = "listItem"
	BlockTable     BlockType = "table"
	BlockImage     BlockType = "image"
	BlockOmitted   BlockType = "omitted"
	BlockPageBreak BlockType = "pageBreak"
)

// Block is one block of a document. Which fields it uses depends on its
// Type, and it marshals to exactly those fields.
type Block struct {
	Type BlockType `json:"type"`
	// Level is a heading's level, 1 to 6, or a list item's nesting, from 0.
	Level int `json:"level,omitempty"`
	// Ordered reports a numbered list item.
	Ordered bool `json:"ordered,omitempty"`
	// Marker is a list item's marker as the document numbers it: "1.",
	// "a)", "•".
	Marker string `json:"marker,omitempty"`
	// Align is a paragraph's alignment, "center", "right" or "justify";
	// empty is the start edge.
	Align string `json:"align,omitempty"`
	// Runs is a heading's, paragraph's or list item's text.
	Runs []Run `json:"runs,omitempty"`
	// Rows is a table's rows.
	Rows []TableRow `json:"rows,omitempty"`
	// Part is an image's part inside the package.
	Part string `json:"part,omitempty"`
	// External reports an image linked from outside the package, which is
	// never fetched; it has no Part.
	External bool `json:"external,omitempty"`
	// Alt is an image's alternative text.
	Alt string `json:"alt,omitempty"`
	// Width and Height are an image's size in CSS pixels, when known.
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`
	// What names what an omitted block stands for: "embedded object",
	// "chart", "equation".
	What string `json:"what,omitempty"`
}

// Run is a stretch of text with one formatting.
type Run struct {
	Text        string `json:"text"`
	Bold        bool   `json:"bold,omitempty"`
	Italic      bool   `json:"italic,omitempty"`
	Underline   bool   `json:"underline,omitempty"`
	Strike      bool   `json:"strike,omitempty"`
	Code        bool   `json:"code,omitempty"`
	Superscript bool   `json:"superscript,omitempty"`
	Subscript   bool   `json:"subscript,omitempty"`
	// Link is the run's hyperlink, only ever http, https or mailto.
	Link string `json:"link,omitempty"`
}

// TableRow is one row of a table.
type TableRow struct {
	Cells []TableCell `json:"cells"`
}

// TableCell is one cell of a table row. A cell a span covers is not listed:
// the cell it belongs to says how many columns and rows it spans.
type TableCell struct {
	ColSpan int     `json:"colSpan"`
	RowSpan int     `json:"rowSpan"`
	Blocks  []Block `json:"blocks"`
}

// FormatForName returns the format a file name's extension says it holds.
func FormatForName(name string) (Format, bool) {
	switch strings.ToLower(path.Ext(name)) {
	case ".docx":
		return FormatDOCX, true
	case ".docm":
		return FormatDOCM, true
	case ".odt":
		return FormatODT, true
	case ".rtf":
		return FormatRTF, true
	}
	return "", false
}

// Convert reads a document and writes it as [ConvertParams.Output] asks. A
// file that cannot be read fails with [ErrMalformed] or [ErrTooLarge]
// wrapped; a document that reaches an output limit is written up to it and
// reported as [ConvertResult.Truncated]. An error from Out is returned as it
// is. A failure after output has begun leaves the stream without its end, so
// a reader of the stream can tell.
func Convert(params ConvertParams) (result ConvertResult, err error) {
	if params.Source == nil || params.Out == nil {
		return ConvertResult{}, errors.New("docreadutil: Convert needs a Source and an Out")
	}
	// Each format's reader.
	var read func(e *emitter, src io.ReaderAt, size int64) error
	switch params.Format {
	case FormatDOCX, FormatDOCM:
		read = readDocx
	case FormatODT:
		read = readOdt
	case FormatRTF:
		read = readRTF
	default:
		return ConvertResult{}, fmt.Errorf("%w: %q", ErrUnsupportedFormat, params.Format)
	}
	e := newEmitter(params)
	defer func() {
		// A reader bug must cost one failed document, never the server.
		if p := recover(); p != nil {
			result, err = ConvertResult{}, fmt.Errorf("%w: the document could not be read: %v", ErrMalformed, p)
		}
	}()
	if err := read(e, params.Source, params.Size); err != nil && !errors.Is(err, errTruncated) {
		return ConvertResult{}, err
	}
	if err := e.finish(); err != nil {
		return ConvertResult{}, err
	}
	return e.result(), nil
}
