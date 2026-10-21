package odputil

// cspell:ignore smil xlink

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/officezip"
)

// The namespaces an ODP reader matches elements and attributes in.
const (
	nsOffice       = "urn:oasis:names:tc:opendocument:xmlns:office:1.0"
	nsStyle        = "urn:oasis:names:tc:opendocument:xmlns:style:1.0"
	nsText         = "urn:oasis:names:tc:opendocument:xmlns:text:1.0"
	nsTable        = "urn:oasis:names:tc:opendocument:xmlns:table:1.0"
	nsDraw         = "urn:oasis:names:tc:opendocument:xmlns:drawing:1.0"
	nsFO           = "urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0"
	nsSVG          = "urn:oasis:names:tc:opendocument:xmlns:svg-compatible:1.0"
	nsPresentation = "urn:oasis:names:tc:opendocument:xmlns:presentation:1.0"
	nsSMIL         = "urn:oasis:names:tc:opendocument:xmlns:smil-compatible:1.0"
	nsAnim         = "urn:oasis:names:tc:opendocument:xmlns:animation:1.0"
	nsXLink        = "http://www.w3.org/1999/xlink"
	nsDC           = "http://purl.org/dc/elements/1.1/"
)

// propPrefixes names the namespaces a style property is kept from, by the
// prefix it is looked up under: "fo:color", "draw:fill". Two namespaces
// share local names — presentation:transition-type is not smil:type — so a
// property is never looked up by its local name alone.
var propPrefixes = map[string]string{
	nsFO:           "fo",
	nsStyle:        "style",
	nsDraw:         "draw",
	nsSVG:          "svg",
	nsPresentation: "presentation",
	nsSMIL:         "smil",
	nsText:         "text",
	nsTable:        "table",
}

// Parts and media types.
const (
	contentPart  = "content.xml"
	stylesPart   = "styles.xml"
	metaPart     = "meta.xml"
	mimePart     = "mimetype"
	manifestPart = "META-INF/manifest.xml"

	mimePresentation = "application/vnd.oasis.opendocument.presentation"
	mimeTemplate     = "application/vnd.oasis.opendocument.presentation-template"
)

// Bounds that are this package's own. officezip bounds what a part may
// unpack to; these bound what the parts held in memory may cost, and what a
// repeat may write that the file does not hold.
const (
	// maxStyleNodes bounds the XML nodes held for styles and master pages,
	// styles.xml and content.xml's automatic styles together.
	maxStyleNodes = 1_000_000
	// maxStyleBytes bounds the text and attribute bytes they hold.
	maxStyleBytes = 32 << 20 // 32 MiB
	// maxPageNodes bounds the XML nodes of one slide, which is held while
	// it is converted.
	maxPageNodes = 250_000
	// maxPageBytes bounds the text and attribute bytes of one slide.
	maxPageBytes = 16 << 20 // 16 MiB
	// maxSlideTextBytes bounds the text one slide writes, text:s expansions
	// included.
	maxSlideTextBytes = 16 << 20 // 16 MiB
	// maxSpaceRun bounds one text:s, which expands to spaces that are not in
	// the file.
	maxSpaceRun = 10_000
	// maxStyleChain bounds how many parent styles are followed, which also
	// ends a cycle.
	maxStyleChain = 32
	// maxTitleBytes bounds the title read from meta.xml.
	maxTitleBytes = 1024
	// maxPolygonPoints bounds one polygon's corners.
	maxPolygonPoints = 10_000
	// The editor's table limits, as pptxutil's import keeps them: a larger
	// table is skipped with a warning.
	maxTableRows    = 500
	maxTableColumns = 100
	maxTableCells   = 5_000
	// importSchemaVersion is the .qslide schema an import writes, the one
	// pptxutil's import writes: every color literal and no theme. Its
	// transitions, tables and shadows are written as the later versions
	// have them, which the editor's migrations carry through unchanged.
	importSchemaVersion = 1
)

// The element types an import writes.
const (
	typeText  = "text"
	typeShape = "shape"
	typeImage = "image"
	typeLine  = "line"
	typeGroup = "group"
	typeTable = "table"
)

// Style scopes. An automatic style's name is unique only within its part, so
// content.xml's "gr1" and styles.xml's "gr1" are different styles: a slide
// looks its automatic styles up in content.xml, a master page in styles.xml.
const (
	scopeContent = iota
	scopeMaster
)

// node is one XML element held whole, or a run of text when name.Local is
// empty. A slide and the styles are held as nodes; nothing else is.
type node struct {
	name     xml.Name
	attrs    []xml.Attr
	children []*node
	text     string
}

// nodeBudget is what may still be held in nodes, and the limit named when it
// runs out.
type nodeBudget struct {
	nodes int
	bytes int
	what  string
}

// props is one properties element of a style, by prefixed name.
type props map[string]string

// odfStyle is one style:style or style:default-style: its parent, the list
// style its paragraphs number with, and its properties by element local
// name ("text-properties", "graphic-properties").
type odfStyle struct {
	parent    string
	listStyle string
	props     map[string]props
}

// listStyle is a text:list-style: per level from 1, whether it numbers or
// bullets.
type listStyle map[int]string

// pageLayout is a style:page-layout's page size in centimeters.
type pageLayout struct {
	width, height float64
}

// masterPage is a style:master-page: its page layout, its drawing-page
// style and its own shapes.
type masterPage struct {
	name      string
	layout    string
	pageStyle string
	shapes    []*node
}

// styleSet is every style the package declares. named holds office:styles
// and the default styles under the name "", auto each scope's automatic
// styles, both keyed by family and name.
type styleSet struct {
	named      map[string]*odfStyle
	auto       [2]map[string]*odfStyle
	lists      [2]map[string]listStyle
	namedLists map[string]listStyle
	fonts      map[string]string
	fillImages map[string]string
	gradients  map[string]string
	layouts    map[string]pageLayout
	masters    []*masterPage
}

// styleRef names a style to look a property up in.
type styleRef struct {
	scope  int
	family string
	name   string
}

// importer carries the state of one import across its slides.
type importer struct {
	pkg    *officezip.Package
	params ImportOdpParams
	styles *styleSet
	// unit is slide units per centimeter.
	unit   float64
	budget *nodeBudget
	// media maps a picture part to the reference it was stored under, ""
	// when it was skipped, and skipped to why.
	media      map[string]string
	skipped    map[string]string
	mediaBytes int64
	elements   int
	nextID     int
	result     ImportOdpResult
}

// slideReader converts one slide's elements, a master's or the slide's own.
type slideReader struct {
	*importer
	scope int
	// master is set while the master page's own shapes are read, and page
	// is the slide's master page, nil when the package has none.
	master    bool
	page      *masterPage
	warnings  []string
	warned    map[string]bool
	textBytes int
}

// textWalker collects the runs of one paragraph, collapsing white space as
// ODF does: a run of spaces, tabs and newlines in the text is one space, and
// none at the start or end of a paragraph.
type textWalker struct {
	s    *slideReader
	runs []outRun
	// cur is the last run's text so far, set on it when the run ends.
	cur       strings.Builder
	lastSpace bool
	started   bool
	err       error
}

// The .qslide an import writes, field for field what pptxutil's import
// writes, which is what quark_slides' QslideCodec.encode writes: the same
// keys in the same order, the same defaults left out, and two-space
// indentation.

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

// outTransition is a slide's transition; the direction is left out when left
// and the duration when 500 ms.
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

// outElement is one of *outText, *outShape, *outImage, *outLine, *outGroup
// and *outTable; box is its frame, for a group to move it into its own.
type outElement interface {
	box() *outFrame
}

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

// outShape is a shape.
type outShape struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Frame        outFrame       `json:"frame"`
	Kind         string         `json:"kind"`
	Preset       string         `json:"preset,omitempty"`
	Contours     []shapeContour `json:"contours,omitempty"`
	Fill         string         `json:"fill,omitempty"`
	Stroke       *outStroke     `json:"stroke,omitempty"`
	CornerRadius *float64       `json:"cornerRadius,omitempty"`
	Shadow       *outShadow     `json:"shadow,omitempty"`
}

// shapeContour is one outline of a path shape in the frame's own space: 0
// at its left or top, 1 at its right or bottom. Fill is set only to false.
type shapeContour struct {
	Commands []pathCommand `json:"commands"`
	Fill     *bool         `json:"fill,omitempty"`
}

// pathCommand is a move, line or close of a contour.
type pathCommand struct {
	Op string
	X  float64
	Y  float64
}

// outShadow is a drop shadow in slide units; the offset and blur are left
// out at 0.
type outShadow struct {
	Color string  `json:"color"`
	X     float64 `json:"x,omitempty"`
	Y     float64 `json:"y,omitempty"`
	Blur  float64 `json:"blur,omitempty"`
}

// outStroke is an outline; dash is left out when solid.
type outStroke struct {
	Color string  `json:"color"`
	Width float64 `json:"width"`
	Dash  string  `json:"dash,omitempty"`
}

// outImage is a picture.
type outImage struct {
	ID      string     `json:"id"`
	Type    string     `json:"type"`
	Frame   outFrame   `json:"frame"`
	Source  string     `json:"source"`
	AltText string     `json:"altText,omitempty"`
	Fit     string     `json:"fit,omitempty"`
	Shadow  *outShadow `json:"shadow,omitempty"`
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
}

// outGroup is a group; Children is never nil and in group-local frames.
type outGroup struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Frame    outFrame     `json:"frame"`
	Children []outElement `json:"children"`
}

// outTable is a table: its frame the sum of its columns and rows, and a
// cell for every row and column.
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

// outCell is one table cell; a blank cell is {}. A span of 1 is left out.
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
// converted, then the close.
type qslideWriter struct {
	w      io.Writer
	slides int
}

func (t *outText) box() *outFrame  { return &t.Frame }
func (s *outShape) box() *outFrame { return &s.Frame }
func (i *outImage) box() *outFrame { return &i.Frame }
func (l *outLine) box() *outFrame  { return &l.Frame }
func (g *outGroup) box() *outFrame { return &g.Frame }
func (t *outTable) box() *outFrame { return &t.Frame }

// MarshalJSON writes only the fields the command's op uses.
func (c pathCommand) MarshalJSON() ([]byte, error) {
	if c.Op == "close" {
		return json.Marshal(map[string]any{"op": c.Op})
	}
	return json.Marshal(map[string]any{"op": c.Op, "x": c.X, "y": c.Y})
}
