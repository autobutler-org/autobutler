package docreadutil

// cspell:ignore ansicpg chftn consolas deff fcharset fixedsys fldinst fldrslt fonttbl fprq ilvl intbl itap listtext lucida menlo nestcell nestrow nesttableprops outlinelevel pagebb pntext striked trowd

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
)

// RTF bounds, beyond the ones [Limits] sets.
const (
	// maxRTFGroupDepth bounds how deeply groups nest, outside a group
	// being skipped, which costs nothing to nest.
	maxRTFGroupDepth = 256
	// maxRTFNameBytes bounds a font's or a style's name, and a list
	// item's marker.
	maxRTFNameBytes = 64
	// maxRTFTitleBytes bounds the title from the \info group.
	maxRTFTitleBytes = 1024
	// maxRTFFieldBytes bounds the instruction text kept of one field,
	// enough for a HYPERLINK with its longest target.
	maxRTFFieldBytes = maxLinkLength + 64
	// rtfContextEvery is how many tokens are read between checks of the
	// conversion's context, so a long stretch of skipped picture data
	// still ends when the request does.
	rtfContextEvery = 1 << 14
)

// Code pages that are not single-byte tables.
const (
	// cpSymbol is a symbol font's: its characters are Word's private use
	// area, U+F000 plus the byte.
	cpSymbol = -1
)

// warnEncoding names text in a double-byte code page written without its
// Unicode form, which the reader does not decode.
const warnEncoding = "Some text in an East Asian encoding is not shown"

// rtfCharacterSets maps \fcharset values to the code page they stand for. A
// charset not listed takes the document's code page.
var rtfCharacterSets = map[int]int{
	0: 1252, 2: cpSymbol, 77: 10000, 128: 932, 129: 949, 130: 1361, 134: 936, 136: 950,
	161: 1253, 162: 1254, 163: 1258, 177: 1255, 178: 1256, 186: 1257, 204: 1251, 222: 874,
	238: 1250, 254: 437, 255: 850,
}

// codeFontNames are the fixed-pitch fonts whose text reads as code when
// the font table does not say it is fixed-pitch.
var codeFontNames = []string{"courier", "consolas", "mono", "menlo", "monaco", "lucida console", "source code", "fixedsys"}

// rtfReader reads one RTF file as blocks, a token at a time.
type rtfReader struct {
	e   *emitter
	lex *rtfLexer
	// stack is the open groups' states, the innermost last; base is the
	// state outside every group, once the root group has closed.
	stack []rtfState
	base  rtfState
	// skipNest counts the groups open inside the group being skipped.
	skipNest int
	tokens   int

	ansiCodePage int
	deff         int
	fonts        map[int]rtfFont
	font         rtfFontEntry
	// styles maps a paragraph style to the heading level it gives, for
	// the styles that give one; styleCount counts every style read.
	styles     map[int]int
	style      rtfStyleEntry
	styleCount int
	title      strings.Builder

	// skip counts the fallback characters still to drop after a \u, and
	// high is a high surrogate waiting for its low half.
	skip int
	high rune
	// lead is a double-byte code page's lead byte, waiting for its trail.
	lead bool

	p      *rtfParagraph
	tables []*rtfTable
	// defs are the row properties at each table nesting level, from 1.
	defs []rtfRowDefs

	// inNote reports a footnote being read, noteDepth the depth of its
	// group, and saved the body paragraph it interrupted.
	inNote    bool
	noteDepth int
	saved     *rtfParagraph
	notes     []Block
	noteBytes int64
	// noteMark is the number of the last note the body marked.
	noteMark int
	// broken reports that the last block placed in the body is a page
	// break, so a paragraph that starts a page does not start another.
	broken bool
}

// readRTF reads an RTF file. RTF is a flat token stream, so it is read
// straight through, never held: a group's state is a few bytes, picture and
// binary data is discarded as it is read, and only a table is held until
// its last row, within the output limit.
func readRTF(e *emitter, src io.ReaderAt, size int64) error {
	r := &rtfReader{
		e:            e,
		lex:          newRTFLexer(io.NewSectionReader(src, 0, size)),
		ansiCodePage: 1252,
		fonts:        map[int]rtfFont{},
		styles:       map[int]int{},
		defs:         make([]rtfRowDefs, e.limits.MaxTableDepth+1),
	}
	if err := r.header(); err != nil {
		return err
	}
	err := r.read()
	if err == nil {
		err = r.finish()
	}
	if errors.Is(err, errTruncated) {
		if saveErr := r.salvage(); saveErr != nil {
			return saveErr
		}
	}
	return err
}

// header checks that the file starts as RTF does, with {\rtf, after at
// most a byte order mark and white space.
func (r *rtfReader) header() error {
	for range 16 {
		tok, err := r.lex.next()
		if err != nil {
			return err
		}
		if tok.kind == rtfByte && (tok.b == ' ' || tok.b == '\t' || tok.b == 0xEF || tok.b == 0xBB || tok.b == 0xBF) {
			continue
		}
		if tok.kind != rtfOpen {
			break
		}
		if tok, err = r.lex.next(); err != nil {
			return err
		}
		if tok.kind != rtfControl || tok.word != wRTF {
			break
		}
		r.stack = append(r.stack, rtfState{uc: 1, char: rtfCharProps{font: -1}, para: defaultRTFPara()})
		return nil
	}
	return fmt.Errorf("%w: the file is not an RTF document", ErrMalformed)
}

// defaultRTFPara is the paragraph formatting \pard restores.
func defaultRTFPara() rtfParaProps {
	return rtfParaProps{outline: -1, itap: -1}
}

// read reads tokens until the root group closes or the file ends.
func (r *rtfReader) read() error {
	for {
		if r.skip == 0 && r.st().dest == destSkip {
			if err := r.lex.skipPlain(); err != nil {
				return err
			}
		}
		tok, err := r.lex.next()
		if err != nil {
			return err
		}
		if r.tokens++; r.tokens%rtfContextEvery == 0 && r.e.ctx != nil {
			if err := r.e.ctx.Err(); err != nil {
				return err
			}
		}
		switch tok.kind {
		case rtfEOF:
			// A file cut off before its last brace reads as far as it goes.
			for len(r.stack) > 0 {
				if _, err := r.close(); err != nil {
					return err
				}
			}
			return nil
		case rtfOpen:
			err = r.open()
		case rtfClose:
			var done bool
			if done, err = r.close(); done {
				return err
			}
		default:
			if r.skip > 0 {
				// A \u's fallback, for readers without Unicode.
				r.skip--
				continue
			}
			if tok.kind == rtfByte {
				err = r.byteText(tok.b)
			} else {
				err = r.control(tok)
			}
		}
		if err != nil {
			return err
		}
	}
}

// st is the innermost group's state.
func (r *rtfReader) st() *rtfState {
	if len(r.stack) == 0 {
		return &r.base
	}
	return &r.stack[len(r.stack)-1]
}

// open opens a group, which starts with its parent's state.
func (r *rtfReader) open() error {
	r.skip, r.lead = 0, false
	parent := r.st()
	if parent.dest == destSkip {
		r.skipNest++
		return nil
	}
	if len(r.stack) >= maxRTFGroupDepth {
		return fmt.Errorf("%w: groups nest deeper than %d", ErrTooLarge, maxRTFGroupDepth)
	}
	st := *parent
	st.starred = false
	switch parent.dest {
	case destFontTable:
		st.dest = destFontEntry
		r.font = rtfFontEntry{id: -1}
	case destStyleSheet:
		st.dest = destStyleEntry
		r.style = rtfStyleEntry{paragraph: true, outline: -1}
	}
	r.stack = append(r.stack, st)
	return nil
}

// close closes a group, finishing what it held. It reports the root
// group's close, the end of the document.
func (r *rtfReader) close() (done bool, err error) {
	r.skip, r.lead = 0, false
	if r.skipNest > 0 {
		r.skipNest--
		return false, nil
	}
	n := len(r.stack)
	if n == 0 {
		return true, nil
	}
	if r.inNote && n == r.noteDepth {
		if err := r.endNote(); err != nil {
			return false, err
		}
	}
	st := r.stack[n-1]
	r.stack = r.stack[:n-1]
	parent := r.st().dest
	switch {
	case st.dest == destFontEntry && parent == destFontTable, st.dest == destFontTable:
		err = r.finishFont()
	case st.dest == destStyleEntry && parent == destStyleSheet:
		err = r.finishStyle()
	case st.dest == destTitle && parent != destTitle:
		r.e.title = strings.Join(strings.Fields(r.title.String()), " ")
	}
	if n == 1 {
		r.base = st
		return true, err
	}
	return false, err
}

// control acts on a control word.
func (r *rtfReader) control(tok rtfToken) error {
	r.lead = false
	st := r.st()
	if st.dest == destSkip {
		return nil
	}
	starred := st.starred
	st.starred = false
	if tok.word == wStar {
		st.starred = true
		return nil
	}
	if handled, err := r.destination(st, tok.word); handled || err != nil {
		return err
	}
	if starred {
		// \* before a destination the reader does not know: skip it.
		st.dest = destSkip
		return nil
	}

	switch st.dest {
	case destFontTable, destFontEntry:
		return r.fontWord(st, tok)
	case destStyleEntry:
		r.styleWord(tok)
		return nil
	case destBody, destTableProps, destListText, destTitle, destFieldInst:
	default:
		return nil
	}
	if r.charWord(st, tok) || r.paraWord(st, tok) {
		return nil
	}
	if handled, err := r.tableWord(st, tok); handled || err != nil {
		return err
	}
	return r.textWord(st, tok)
}

// destination acts on a control word that starts a destination, and
// reports whether it was one.
func (r *rtfReader) destination(st *rtfState, word rtfWord) (bool, error) {
	body := st.dest == destBody
	switch word {
	case wFontTbl:
		st.dest = destFontTable
		r.font = rtfFontEntry{id: -1}
	case wStyleSheet:
		st.dest = destStyleSheet
	case wInfo:
		st.dest = destInfo
	case wTitle:
		if st.dest != destInfo {
			st.dest = destSkip
			break
		}
		st.dest = destTitle
		r.title.Reset()
	case wField:
		if body {
			st.field = &rtfField{}
		}
	case wFldInst:
		if st.field == nil || !body {
			st.dest = destSkip
			break
		}
		st.dest = destFieldInst
	case wFldRslt:
		if st.field != nil && body {
			if link := st.field.hyperlink(); link != "" {
				st.link = link
			}
		}
	case wListText, wPnText:
		if !body {
			st.dest = destSkip
			break
		}
		p, err := r.para()
		if err != nil {
			return true, err
		}
		st.dest = destListText
		p.marker.Reset()
		p.hasMarker = true
	case wFootnote:
		if !body || r.inNote {
			st.dest = destSkip
			break
		}
		r.startNote()
	case wShpPict:
		// A picture's container, read for the \pict inside it.
	case wNestTableProps:
		st.dest = destTableProps
	case wPict, wObject, wShape, wMath:
		st.dest = destSkip
		if body {
			what := map[rtfWord]string{wPict: "image", wObject: "embedded object", wShape: "shape", wMath: "equation"}[word]
			r.e.warnings.count(what)
			return true, r.split(Block{Type: BlockOmitted, What: what})
		}
	case wHeaderFooter:
		st.dest = destSkip
		r.e.warnings.note(warnHeadersFooters)
	case wAnnotation:
		st.dest = destSkip
		r.e.warnings.note(warnComments)
	case wSkipDest:
		st.dest = destSkip
	default:
		return false, nil
	}
	return true, nil
}

// fontWord acts on a control word inside the font table.
func (r *rtfReader) fontWord(st *rtfState, tok rtfToken) error {
	switch tok.word {
	case wF:
		if st.dest == destFontTable {
			// The table's entries without a group each, ended by ";".
			if err := r.finishFont(); err != nil {
				return err
			}
		}
		r.font.id = max(tok.param, 0)
	case wFCharset:
		if cp, ok := rtfCharacterSets[tok.param]; ok && r.font.font.codePage == 0 {
			r.font.font.codePage = cp
		}
	case wCpg:
		if _, ok := singleByteCodePages[tok.param]; ok || isDoubleByte(tok.param) {
			r.font.font.codePage = tok.param
		}
	case wFPrq:
		r.font.font.code = r.font.font.code || tok.param == 1
	}
	return nil
}

// finishFont records the font table entry read so far.
func (r *rtfReader) finishFont() error {
	entry := &r.font
	if entry.id < 0 {
		return nil
	}
	if _, seen := r.fonts[entry.id]; !seen && len(r.fonts) >= r.e.limits.MaxDefinitions {
		return fmt.Errorf("%w: the font table has more than %d fonts", ErrTooLarge, r.e.limits.MaxDefinitions)
	}
	name := strings.ToLower(entry.name.String())
	for _, code := range codeFontNames {
		if strings.Contains(name, code) {
			entry.font.code = true
		}
	}
	r.fonts[entry.id] = entry.font
	r.font = rtfFontEntry{id: -1}
	return nil
}

// styleWord acts on a control word inside a style sheet entry.
func (r *rtfReader) styleWord(tok rtfToken) {
	switch tok.word {
	case wS:
		r.style.id = tok.param
	case wCharStyle:
		r.style.paragraph = false
	case wOutlineLevel:
		r.style.outline = tok.param
	}
}

// finishStyle records the style sheet entry just read, if it is a
// paragraph style that makes a heading.
func (r *rtfReader) finishStyle() error {
	if r.styleCount++; r.styleCount > r.e.limits.MaxDefinitions {
		return fmt.Errorf("%w: the style sheet has more than %d styles", ErrTooLarge, r.e.limits.MaxDefinitions)
	}
	if !r.style.paragraph {
		return nil
	}
	// A name ends at its ";", and may list aliases after a comma.
	name, _, _ := strings.Cut(r.style.name.String(), ";")
	name, _, _ = strings.Cut(name, ",")
	level := headingForName(name)
	if r.style.outline >= 0 {
		level = headingForOutline(r.style.outline)
	}
	if level > 0 {
		r.styles[r.style.id] = level
	} else {
		delete(r.styles, r.style.id)
	}
	return nil
}

// on is a toggle control word's value: on, unless its parameter is 0.
func on(tok rtfToken) bool { return !tok.hasParam || tok.param != 0 }

// charWord acts on a character formatting control word, and reports
// whether it was one.
func (r *rtfReader) charWord(st *rtfState, tok rtfToken) bool {
	c := &st.char
	switch tok.word {
	case wPlain:
		*c = rtfCharProps{font: -1}
	case wB:
		c.bold = on(tok)
	case wI:
		c.italic = on(tok)
	case wUl:
		c.underline = on(tok)
	case wUlNone:
		c.underline = false
	case wStrike:
		c.strike = on(tok)
	case wStrikeD:
		c.strike = tok.param != 0
	case wSuper:
		c.sup, c.sub = on(tok), false
	case wSub:
		c.sub, c.sup = on(tok), false
	case wNoSuperSub:
		c.sup, c.sub = false, false
	case wUp:
		// \upN and \dnN raise and lower text by N half-points, as
		// LibreOffice writes superscript and subscript.
		c.sup, c.sub = on(tok), false
	case wDn:
		c.sub, c.sup = on(tok), false
	case wV:
		c.hidden = on(tok)
	case wDeleted:
		c.deleted = on(tok)
	case wF:
		c.font = max(tok.param, 0)
	case wUC:
		st.uc = min(max(tok.param, 0), 16)
	default:
		return false
	}
	return true
}

// paraWord acts on a paragraph formatting control word, and reports
// whether it was one.
func (r *rtfReader) paraWord(st *rtfState, tok rtfToken) bool {
	p := &st.para
	switch tok.word {
	case wPard:
		*p = defaultRTFPara()
	case wS:
		p.style = tok.param
	case wOutlineLevel:
		p.outline = tok.param
	case wQL:
		p.align = ""
	case wQC:
		p.align = "center"
	case wQR:
		p.align = "right"
	case wQJ:
		p.align = "justify"
	case wInTbl:
		p.intbl = on(tok)
	case wItap:
		p.itap = max(tok.param, 0)
	case wIlvl:
		p.ilvl = min(max(tok.param, 0), 8)
	case wLs:
		p.listed = tok.param > 0
	case wPageBB:
		p.pageBreakBefore = on(tok)
	case wAnsi:
		r.ansiCodePage = 1252
	case wMac:
		r.ansiCodePage = 10000
	case wPC:
		r.ansiCodePage = 437
	case wPCA:
		r.ansiCodePage = 850
	case wAnsiCpg:
		if _, ok := singleByteCodePages[tok.param]; ok || isDoubleByte(tok.param) {
			r.ansiCodePage = tok.param
		}
	case wDeff:
		r.deff = max(tok.param, 0)
	default:
		return false
	}
	return true
}

// textWord acts on a control word that is text or ends some: breaks,
// special characters, \u and note marks.
func (r *rtfReader) textWord(st *rtfState, tok rtfToken) error {
	if ch, ok := rtfCharacters[tok.word]; ok {
		return r.emit(st, ch)
	}
	switch tok.word {
	case wU:
		return r.unicode(st, tok.param)
	case wTab:
		return r.emit(st, '\t')
	case wLine:
		return r.emit(st, '\n')
	}
	if st.dest != destBody {
		return nil
	}
	switch tok.word {
	case wPar:
		return r.endParagraph(r.level(st.para))
	case wSect:
		if r.p != nil {
			return r.endParagraph(r.level(st.para))
		}
	case wPage:
		return r.pageBreak(st)
	case wChftn:
		if !r.inNote {
			r.noteMark++
		}
		if r.noteMark > 0 {
			mark := *st
			mark.char.sup, mark.char.sub, mark.link = true, false, ""
			return r.text(&mark, fmt.Sprint(r.noteMark))
		}
	}
	return nil
}

// unicode emits the character a \u names, and starts skipping its
// fallback. A character beyond the Basic Multilingual Plane comes as two
// \u, a surrogate pair.
func (r *rtfReader) unicode(st *rtfState, n int) error {
	r.skip = st.uc
	if n < 0 {
		n += 0x10000
	}
	ch, high := rune(n), r.high
	r.high = 0
	switch {
	case ch >= 0xD800 && ch <= 0xDBFF:
		r.high = ch
		return nil
	case ch >= 0xDC00 && ch <= 0xDFFF:
		if high == 0 {
			return nil
		}
		ch = 0x10000 + (high-0xD800)<<10 + (ch - 0xDC00)
	case ch < 0 || ch > unicode.MaxRune:
		return nil
	}
	return r.emit(st, ch)
}

// byteText acts on a byte of text, decoded in the code page of the font it
// is in.
func (r *rtfReader) byteText(b byte) error {
	st := r.st()
	st.starred = false
	switch st.dest {
	case destFontTable, destFontEntry:
		if b == ';' {
			return r.finishFont()
		}
		if r.font.id >= 0 && b < 0x80 && r.font.name.Len() < maxRTFNameBytes {
			r.font.name.WriteByte(b)
		}
		return nil
	case destStyleEntry:
		if b < 0x80 && r.style.name.Len() < maxRTFNameBytes {
			r.style.name.WriteByte(b)
		}
		return nil
	case destBody, destTitle, destFieldInst, destListText:
	default:
		return nil
	}
	ch, ok := r.decode(st, b)
	if !ok {
		return nil
	}
	return r.emit(st, ch)
}

// decode turns a byte into the character its font's code page gives it.
func (r *rtfReader) decode(st *rtfState, b byte) (rune, bool) {
	cp := r.codePage(st.char.font)
	if isDoubleByte(cp) {
		if r.lead {
			// The pair's character is not decoded; see warnEncoding.
			r.lead = false
			r.e.warnings.note(warnEncoding)
			return 0, false
		}
		if isLeadByte(cp, b) {
			r.lead = true
			return 0, false
		}
		if cp == 932 && b >= 0xA1 && b <= 0xDF {
			// Shift JIS half-width katakana.
			return 0xFF61 + rune(b-0xA1), true
		}
	}
	if b < 0x80 {
		return rune(b), true
	}
	if cp == cpSymbol {
		return 0xF000 + rune(b), true
	}
	table, ok := singleByteCodePages[cp]
	if !ok {
		table = singleByteCodePages[1252]
	}
	ch := table[b-0x80]
	return ch, ch != 0
}

// codePage is the code page text in a font is written in.
func (r *rtfReader) codePage(font int) int {
	if font < 0 {
		font = r.deff
	}
	if f, ok := r.fonts[font]; ok && f.codePage != 0 {
		return f.codePage
	}
	return r.ansiCodePage
}

// isDoubleByte reports one of the East Asian code pages, whose characters
// take one byte or two.
func isDoubleByte(cp int) bool {
	switch cp {
	case 932, 936, 949, 950, 1361:
		return true
	}
	return false
}

// isLeadByte reports a byte that starts a two-byte character in a
// double-byte code page.
func isLeadByte(cp int, b byte) bool {
	if cp == 932 {
		return b >= 0x81 && b <= 0x9F || b >= 0xE0 && b <= 0xFC
	}
	return b >= 0x81 && b <= 0xFE
}

// text emits each character of s.
func (r *rtfReader) text(st *rtfState, s string) error {
	for _, ch := range s {
		if err := r.emit(st, ch); err != nil {
			return err
		}
	}
	return nil
}

// emit sends a character to where the group's text goes.
func (r *rtfReader) emit(st *rtfState, ch rune) error {
	if ch >= 0xF000 && ch <= 0xF0FF && st.dest != destListText {
		// A symbol font's character, which no other font draws; its bullet
		// is the one most often typed.
		if ch != 0xF0B7 {
			return nil
		}
		ch = 0x2022
	}
	if ch < 0x20 && ch != '\t' && ch != '\n' || ch == 0x7F {
		return nil
	}
	switch st.dest {
	case destBody:
		return r.appendRune(st, ch)
	case destTitle:
		if r.title.Len() < maxRTFTitleBytes {
			r.title.WriteRune(ch)
		}
	case destFieldInst:
		if st.field.inst.Len() < maxRTFFieldBytes {
			st.field.inst.WriteRune(ch)
		}
	case destListText:
		if r.p != nil && ch != '\t' && ch != '\n' && r.p.marker.Len() < maxRTFNameBytes {
			r.p.marker.WriteRune(ch)
		}
	}
	return nil
}

// appendRune adds a character to the paragraph being read, in the group's
// formatting. Hidden text and tracked deletions are dropped: tracked
// changes read as accepted.
func (r *rtfReader) appendRune(st *rtfState, ch rune) error {
	c := st.char
	if c.hidden || c.deleted {
		return nil
	}
	p, err := r.para()
	if err != nil {
		return err
	}
	font := c.font
	if font < 0 {
		font = r.deff
	}
	run := Run{
		Bold: c.bold, Italic: c.italic, Underline: c.underline, Strike: c.strike,
		Code: r.fonts[font].code, Superscript: c.sup, Subscript: c.sub && !c.sup, Link: st.link,
	}
	cost := 4
	if !p.open || p.cur != run {
		cost += runOverhead
	}
	if err := r.e.charge(cost); err != nil {
		return err
	}
	if p.open && p.cur != run {
		p.commit()
	}
	p.cur, p.open = run, true
	p.text.WriteRune(ch)
	return nil
}

// para returns the paragraph being read, starting one if none is.
func (r *rtfReader) para() (*rtfParagraph, error) {
	if r.p != nil {
		return r.p, nil
	}
	if err := r.e.admit(); err != nil {
		return nil, err
	}
	if err := r.e.charge(blockOverhead); err != nil {
		return nil, err
	}
	r.p = &rtfParagraph{}
	r.p.block.Type, r.p.admitted = BlockParagraph, true
	return r.p, nil
}

// level is the table nesting level a paragraph sits at: 0 for the body,
// and no deeper than the table depth limit, past which tables flatten into
// the deepest cell allowed. A note's paragraphs are never in a table.
func (r *rtfReader) level(pp rtfParaProps) int {
	if r.inNote {
		return 0
	}
	level := max(pp.itap, 0)
	if pp.intbl && level == 0 {
		level = 1
	}
	return min(level, r.e.limits.MaxTableDepth)
}

// shape settles a paragraph's block type from its formatting: a heading
// from its outline level or style, a list item from its marker or list.
func (r *rtfReader) shape(p *rtfParagraph, pp rtfParaProps) {
	if p.shaped {
		return
	}
	p.shaped = true
	p.block.Align = pp.align
	heading := r.styles[pp.style]
	if pp.outline >= 0 {
		heading = headingForOutline(pp.outline)
	}
	switch {
	case heading > 0:
		p.block.Type, p.block.Level = BlockHeading, heading
	case p.hasMarker || pp.listed:
		marker := strings.TrimSpace(p.marker.String())
		bullet := bulletMarker(marker)
		ordered := bullet == marker && strings.IndexFunc(marker, func(ch rune) bool {
			return unicode.IsLetter(ch) || unicode.IsDigit(ch)
		}) >= 0
		if !ordered {
			marker = bullet
		}
		p.block.Type, p.block.Level, p.block.Ordered, p.block.Marker = BlockListItem, pp.ilvl, ordered, marker
	}
}

// endParagraph places the paragraph being read at a table level. A \par
// with nothing before it is an empty paragraph, which is vertical space in
// the document, unless a split has already placed part of it.
func (r *rtfReader) endParagraph(level int) error {
	p, err := r.para()
	if err != nil {
		return err
	}
	pp := r.st().para
	p.commit()
	r.shape(p, pp)
	if len(p.block.Runs) == 0 && p.placed {
		r.p = nil
		return nil
	}
	if !p.admitted {
		if err := r.e.admit(); err != nil {
			return err
		}
		p.admitted = true
	}
	if pp.pageBreakBefore && level == 0 && !r.inNote && !p.placed && !r.broken {
		p.placed = true
		if err := r.place(Block{Type: BlockPageBreak}, 0); err != nil {
			return err
		}
	}
	if err := r.place(p.block, level); err != nil {
		return err
	}
	r.p = nil
	return nil
}

// split places b where the document has it, first placing what the
// paragraph being read has collected so far. What follows the split
// continues as a plain paragraph, so a list item's marker or a heading is
// not repeated.
func (r *rtfReader) split(b Block) error {
	level := r.level(r.st().para)
	if p := r.p; p != nil {
		p.commit()
		if len(p.block.Runs) > 0 {
			r.shape(p, r.st().para)
			if !p.admitted {
				if err := r.e.admit(); err != nil {
					return err
				}
			}
			placed := p.block
			p.block = Block{Type: BlockParagraph, Align: placed.Align}
			p.admitted, p.placed = false, true
			if err := r.place(placed, level); err != nil {
				return err
			}
		}
	}
	if err := r.e.admit(); err != nil {
		return err
	}
	if err := r.e.charge(blockOverhead); err != nil {
		return err
	}
	return r.place(b, level)
}

// pageBreak places a page break. One inside a table or a note has nowhere
// to go and is dropped.
func (r *rtfReader) pageBreak(st *rtfState) error {
	if r.inNote || r.level(st.para) > 0 {
		return nil
	}
	return r.split(Block{Type: BlockPageBreak})
}

// place puts a finished block at a table level: into a note, into the
// cell being read at that level, or out to the writer.
func (r *rtfReader) place(b Block, level int) error {
	if r.inNote {
		return r.noteBlock(b)
	}
	r.e.tally(b)
	if err := r.toDepth(level); err != nil {
		return err
	}
	r.broken = b.Type == BlockPageBreak
	if n := len(r.tables); n > 0 {
		t := r.tables[n-1]
		t.cur = append(t.cur, b)
		return nil
	}
	return r.e.write(b)
}

// startNote starts reading a footnote: its paragraphs are held, to follow
// the body, and the body paragraph it interrupts waits for it to end.
func (r *rtfReader) startNote() {
	r.inNote, r.noteDepth = true, len(r.stack)
	r.saved, r.p = r.p, nil
}

// endNote ends the footnote being read, before its group closes, and
// returns to the body paragraph it interrupted.
func (r *rtfReader) endNote() error {
	if r.p != nil {
		if err := r.endParagraph(0); err != nil {
			return err
		}
	}
	r.inNote = false
	r.p, r.saved = r.saved, nil
	return nil
}

// noteBlock holds a note's block, to be written after the body. Notes past
// what may be held are left out, with a warning.
func (r *rtfReader) noteBlock(b Block) error {
	cost := blockCost(b)
	if len(r.notes) >= r.e.limits.MaxDefinitions || r.noteBytes+cost > r.e.limits.MaxHeldBytes {
		r.e.warnings.note(warnNotes)
		return nil
	}
	r.e.tally(b)
	r.notes = append(r.notes, b)
	r.noteBytes += cost
	return nil
}

// blockCost estimates what a text block costs in the output.
func blockCost(b Block) int64 {
	cost := int64(blockOverhead + len(b.Marker))
	for _, run := range b.Runs {
		cost += int64(runOverhead + len(run.Text) + len(run.Link))
	}
	return cost
}

// finish places what the end of the file leaves open, then the notes
// under one "Notes" heading.
func (r *rtfReader) finish() error {
	if r.p != nil {
		if err := r.endParagraph(r.level(r.st().para)); err != nil {
			return err
		}
	}
	if err := r.toDepth(0); err != nil {
		return err
	}
	if len(r.notes) == 0 {
		return nil
	}
	if err := r.e.admit(); err != nil {
		return err
	}
	// The heading is Quark's, not the document's, so it is not counted.
	if err := r.e.write(Block{Type: BlockHeading, Level: 1, Runs: []Run{{Text: "Notes"}}}); err != nil {
		return err
	}
	for _, b := range r.notes {
		if err := r.e.charge(int(blockCost(b))); err != nil {
			return err
		}
		if err := r.e.write(b); err != nil {
			return err
		}
	}
	return nil
}

// salvage places what a limit cut short: the paragraph being read, and
// the tables open around it, with the rows they have.
func (r *rtfReader) salvage() error {
	if r.inNote {
		// The note is dropped; the body paragraph it interrupted is not.
		r.inNote, r.p, r.saved = false, r.saved, nil
	}
	if p := r.p; p != nil {
		r.p = nil
		p.commit()
		if len(p.block.Runs) > 0 && p.admitted {
			r.shape(p, r.st().para)
			r.e.tally(p.block)
			if n := len(r.tables); n > 0 {
				r.tables[n-1].cur = append(r.tables[n-1].cur, p.block)
			} else if err := r.e.write(p.block); err != nil {
				return err
			}
		}
	}
	for len(r.tables) > 0 {
		if err := r.closeTable(); err != nil {
			return err
		}
	}
	return nil
}
