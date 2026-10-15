package docreadutil

// cspell:ignore ilvl Ilvl

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/officezip"
)

// blockWriter is one of the two writers a block stream goes to. begin is
// called once, before the first block; end once, after the last.
type blockWriter interface {
	begin(meta metaLine) error
	block(b Block) error
	end(end endLine) error
	// losses names what the writer could not hold, as warnings.
	losses() []string
}

// metaLine is the first line of the NDJSON stream.
type metaLine struct {
	Type    string `json:"type"`
	Version int    `json:"version"`
	Format  Format `json:"format"`
	Title   string `json:"title,omitempty"`
}

// endLine is the last line of the NDJSON stream.
type endLine struct {
	Type       string   `json:"type"`
	Truncated  bool     `json:"truncated"`
	Words      int      `json:"words"`
	Characters int      `json:"characters"`
	Paragraphs int      `json:"paragraphs"`
	Warnings   []string `json:"warnings"`
}

// countingWriter counts what passes through it to w.
type countingWriter struct {
	w io.Writer
	n int64
}

// emitter takes the blocks a reader reads, enforces the output limits, keeps
// the counts and warnings, and hands top-level blocks to the writer.
type emitter struct {
	ctx    context.Context
	limits Limits
	format Format
	out    *countingWriter
	w      blockWriter
	title  string
	// started is set once the writer has begun.
	started bool
	// blocks counts every block admitted, nested ones included.
	blocks int
	// held estimates the bytes of what is read but not yet written: a
	// paragraph being read, or a table not yet finished.
	held      int64
	truncated bool
	words     int
	chars     int
	paras     int
	warnings  warnings
}

// warnings collects what a reader left out, counted by kind, in the order
// each kind was first seen.
type warnings struct {
	order  []string
	counts map[string]int
}

// The wire shapes of each block type, so each marshals to exactly the
// fields it uses.
type (
	headingJSON struct {
		Type  BlockType `json:"type"`
		Level int       `json:"level"`
		Runs  []Run     `json:"runs"`
	}
	paragraphJSON struct {
		Type  BlockType `json:"type"`
		Align string    `json:"align,omitempty"`
		Runs  []Run     `json:"runs"`
	}
	listItemJSON struct {
		Type    BlockType `json:"type"`
		Ordered bool      `json:"ordered"`
		Level   int       `json:"level"`
		Marker  string    `json:"marker"`
		Align   string    `json:"align,omitempty"`
		Runs    []Run     `json:"runs"`
	}
	tableJSON struct {
		Type BlockType  `json:"type"`
		Rows []TableRow `json:"rows"`
	}
	imageJSON struct {
		Type     BlockType `json:"type"`
		Part     string    `json:"part,omitempty"`
		External bool      `json:"external,omitempty"`
		Alt      string    `json:"alt,omitempty"`
		Width    int       `json:"width,omitempty"`
		Height   int       `json:"height,omitempty"`
	}
	omittedJSON struct {
		Type BlockType `json:"type"`
		What string    `json:"what"`
	}
	typeJSON struct {
		Type BlockType `json:"type"`
	}
)

// ndjsonWriter writes the viewer's stream, one JSON object per line.
type ndjsonWriter struct {
	enc *json.Encoder
}

// deltaWriter writes a .qdoc, a Quill Delta, one op at a time.
type deltaWriter struct {
	w io.Writer
	// ops counts the ops written.
	ops int
	// tables and images count what was flattened or left out.
	tables, images int
}

// deltaOp is one Quill insert.
type deltaOp struct {
	Insert     string      `json:"insert"`
	Attributes *deltaAttrs `json:"attributes,omitempty"`
}

// deltaAttrs are the inline and line attributes a .qdoc understands.
type deltaAttrs struct {
	Bold      bool   `json:"bold,omitempty"`
	Italic    bool   `json:"italic,omitempty"`
	Underline bool   `json:"underline,omitempty"`
	Strike    bool   `json:"strike,omitempty"`
	Code      bool   `json:"code,omitempty"`
	Script    string `json:"script,omitempty"`
	Link      string `json:"link,omitempty"`
	Header    int    `json:"header,omitempty"`
	List      string `json:"list,omitempty"`
	Indent    int    `json:"indent,omitempty"`
	Align     string `json:"align,omitempty"`
}

// tokenSource is an XML token stream: officezip's counting decoder for a
// streamed part, or a held part's decoder.
type tokenSource interface {
	Token() (xml.Token, error)
	Skip() error
}

// depthCounter passes tokens through from a held part's decoder and fails
// once its elements nest deeper than max, before the decoder's own stack of
// open elements can grow past it.
type depthCounter struct {
	dec   *xml.Decoder
	name  string
	max   int
	depth int
	err   error
}

// capReader reads at most n bytes of a held part and fails the read that
// would need one more.
type capReader struct {
	r    io.Reader
	name string
	n    int64
	read int64
}

// opt is a run property a style or a run may set on, set off, or leave to
// what it is based on.
type opt uint8

const (
	optUnset opt = iota
	optOn
	optOff
)

// runProps is the character formatting a run, a character style or a
// paragraph style sets.
type runProps struct {
	bold, italic, underline, strike, code, sup, sub, hidden opt
}

// paraProps is what a paragraph's w:pPr, or a style's, sets.
type paraProps struct {
	style string
	// outline is the outline level, 0 to 9, or -1 when unset.
	outline int
	// numID and ilvl are the list, when hasNum; numID "0" removes one.
	numID           string
	ilvl            int
	hasNum, hasIlvl bool
	align           string
	pageBreakBefore bool
}

// docxStyle is one w:style of styles.xml, as declared.
type docxStyle struct {
	kind, basedOn, name string
	para                paraProps
	run                 runProps
}

// resolvedStyle is a style with everything it is based on applied.
type resolvedStyle struct {
	run     runProps
	heading int
	para    paraProps
}

// docxStyles is styles.xml, held.
type docxStyles struct {
	byID map[string]*docxStyle
	// defaults is w:docDefaults' run formatting.
	defaults runProps
	resolved map[string]*resolvedStyle
	// defaultPara is the style a paragraph without w:pStyle takes.
	defaultPara string
}

// numLevel is one w:lvl of a list definition.
type numLevel struct {
	format, text string
	start        int
}

// numLevels is a list's nine levels; a nil level is undefined.
type numLevels [9]*numLevel

// numDef is one w:num: the abstract list it instantiates and the levels it
// overrides.
type numDef struct {
	abstract  string
	overrides numLevels
	// starts are w:startOverride values, -1 when unset.
	starts [9]int
}

// listCounter is where one list's numbering has got to.
type listCounter struct {
	value   [9]int
	started [9]bool
}

// docxNumbering is numbering.xml, held, with the counters of every list
// the document has numbered so far.
type docxNumbering struct {
	abstract map[string]*numLevels
	lists    map[string]*numDef
	counters map[string]*listCounter
}

// noteKind is a footnote or an endnote.
type noteKind int

const (
	footnote noteKind = iota
	endnote
)

// noteMarks numbers the notes of one kind in the order the body refers to
// them.
type noteMarks struct {
	marks map[string]string
	next  int
}

// docxReader reads one DOCX package's body as blocks.
type docxReader struct {
	pkg       *officezip.Package
	e         *emitter
	src       tokenSource
	styles    *docxStyles
	numbering *docxNumbering
	// part is the part being read, and rels its relationships by id.
	part string
	rels map[string]officezip.Relationship
	// cell is the cell whose blocks are being read; nil is the top level.
	cell       *[]Block
	tableDepth int
	// fields is one entry per open complex field, true once its result has
	// begun; text inside any field's instructions is dropped.
	fields []bool
	notes  [2]noteMarks
	// note is the mark of the note being read, which its w:footnoteRef
	// shows.
	note string
}

// paragraph collects one w:p's runs into a block.
type paragraph struct {
	block Block
	// base is the run formatting its style gives every run.
	base runProps
	// admitted reports that the block counts against the block limit;
	// placed that part of the paragraph has been placed already, before
	// an image or a page break split it.
	admitted, placed bool
	// cur is the formatting of the run being collected, while open, and
	// text its text so far.
	cur  Run
	open bool
	text strings.Builder
}

// tableBuilder collects one table's rows.
type tableBuilder struct {
	rows []TableRow
	// merging maps a grid column to the cell a vertical merge started
	// there, while the merge runs.
	merging map[int]cellIndex
}

// cellIndex locates a cell in a tableBuilder.
type cellIndex struct {
	row, cell int
}

// drawingInfo is what a w:drawing, w:pict or w:object says about itself.
type drawingInfo struct {
	alt           string
	width, height int
	// embed and link are a picture's relationship ids.
	embed, link string
	// what is the omitted kind when the drawing is not a picture.
	what string
}
