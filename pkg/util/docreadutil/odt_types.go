package docreadutil

import "github.com/autobutler-org/quark/pkg/util/officezip"

// odtLevels is how many list levels ODF defines.
const odtLevels = 10

// odtStyle is one style:style or style:default-style, as declared.
type odtStyle struct {
	parent    string
	listStyle string
	// heading is the heading level the style's name gives, 0 for none.
	heading int
	// outline is style:default-outline-level, 0 when unset.
	outline int
	run     runProps
	// align is the paragraph alignment, when alignSet.
	align                 string
	alignSet              bool
	breakBefore, breakAft opt
}

// odtResolved is a style with its parents and the family's default style
// applied.
type odtResolved struct {
	run         runProps
	heading     int
	outline     int
	align       string
	listStyle   string
	breakBefore bool
	breakAfter  bool
}

// odtListLevel is one level of a text:list-style.
type odtListLevel struct {
	// numbered is a text:list-level-style-number; otherwise the level is a
	// bullet, or an image shown as one.
	numbered bool
	// bullet is a bullet level's character.
	bullet string
	// format is style:num-format: "1", "a", "A", "i", "I", or empty for a
	// level that shows no number.
	format         string
	prefix, suffix string
	start          int
	// display is how many levels the marker shows, this one and those
	// above it.
	display    int
	letterSync bool
}

// odtListStyle is a text:list-style's levels; a nil level is undefined.
type odtListStyle [odtLevels]*odtListLevel

// odtStyles is every style a document declares: styles.xml's, held, and
// content.xml's automatic styles, read as the part streams past.
type odtStyles struct {
	limits Limits
	// common and automatic are styles by family and name. A name looks in
	// automatic first; a parent only ever names a common style.
	common, automatic map[string]*odtStyle
	// defaults is each family's style:default-style.
	defaults map[string]*odtStyle
	lists    map[string]*odtListStyle
	// fixed is the font faces of fixed pitch, by name.
	fixed    map[string]bool
	resolved map[string]*odtResolved
	// count and bytes are what the definitions so far have cost.
	count int
	bytes int64
}

// odtCounter is where one list's numbering has got to.
type odtCounter struct {
	// style is the list style of the list that began the numbering, which
	// a list continuing it without naming its own takes.
	style   string
	value   [odtLevels]int
	started [odtLevels]bool
}

// odtItem is a text:list-item whose first paragraph has not been read yet.
type odtItem struct {
	// start is text:start-value, empty when unset.
	start string
}

// odtReader reads one ODT package's body as blocks.
type odtReader struct {
	blockPlacer
	pkg    *officezip.Package
	src    tokenSource
	styles *odtStyles
	// objects is the media type of each embedded object, by its directory
	// in the package.
	objects map[string]string
	// listDepth is how many text:list elements are open, listStyle the list
	// style they use, and counter the numbering of the outermost.
	listDepth int
	listStyle string
	counter   *odtCounter
	// item is the list item waiting for its first paragraph.
	item *odtItem
	// byStyle is the numbering of the last list of each style, which
	// text:continue-numbering continues, and byID that of each list by its
	// xml:id, which text:continue-list names.
	byStyle, byID map[string]*odtCounter
	// notes is the mark each cited note shows, by its text:id.
	notes map[string]string
	// note is the mark of the note being read, which its first paragraph
	// shows.
	note string
	// space reports that the paragraph being read ends in white space, or
	// has no text yet, so white space that follows collapses into it.
	space bool
}
