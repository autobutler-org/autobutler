package docreadutil

// cspell:ignore ilvl altChunk bdo delText endnote endnoteRef endnoteReference endnotes fldChar fldCharType fldSimple footnoteRef footnoteReference footnotes gridBefore gridSpan instrText moveFrom moveTo noBreakHyphen oMath oMathPara pPr ptab rPr rubyBase sdt sdtContent tbl tc tcPr tr trPr vMerge

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/officezip"
)

// Relationship types, matched by their last segment, which transitional and
// strict OOXML share.
const (
	relOfficeDocument = "/officeDocument"
	relCoreProperties = "/core-properties"
	relStyles         = "/styles"
	relNumbering      = "/numbering"
	relFootnotes      = "/footnotes"
	relEndnotes       = "/endnotes"
	relHeader         = "/header"
	relFooter         = "/footer"
	relComments       = "/comments"
)

// defaultDocumentPart is where a package without relationships keeps its
// body.
const defaultDocumentPart = "word/document.xml"

// Estimates of what each piece of a block costs in the output, beyond its
// text, charged while it is held.
const (
	blockOverhead = 32
	runOverhead   = 16
	rowOverhead   = 12
	cellOverhead  = 40
)

// maxFieldDepth bounds how many complex fields nest; deeper ones are
// counted but not tracked.
const maxFieldDepth = 64

// maxLinkLength bounds a hyperlink's target.
const maxLinkLength = 2048

// readDocx reads a DOCX or DOCM package's body, then its notes.
func readDocx(e *emitter, src io.ReaderAt, size int64) error {
	pkg, err := officezip.Open(src, size, e.limits.Zip)
	if err != nil {
		return err
	}
	pkgRels, err := pkg.Relationships("")
	if err != nil {
		return err
	}
	main := defaultDocumentPart
	if rel, ok := internalRel(pkgRels, relOfficeDocument); ok {
		main = rel.Part
	}
	if !pkg.Has(main) {
		return fmt.Errorf("%w: the package holds no word-processing document", ErrMalformed)
	}
	if rel, ok := internalRel(pkgRels, relCoreProperties); ok && pkg.Has(rel.Part) {
		e.title = readTitle(pkg, rel.Part, e.limits.MaxHeldBytes, "coreProperties", "")
	}
	mainRels, err := pkg.Relationships(main)
	if err != nil {
		return err
	}

	r := &docxReader{blockPlacer: blockPlacer{e: e}, pkg: pkg, styles: emptyStyles(), numbering: emptyNumbering()}
	if rel, ok := internalRel(mainRels, relStyles); ok && pkg.Has(rel.Part) {
		if r.styles, err = readStyles(pkg, rel.Part, e.limits); err != nil {
			return err
		}
	}
	if rel, ok := internalRel(mainRels, relNumbering); ok && pkg.Has(rel.Part) {
		if r.numbering, err = readNumbering(pkg, rel.Part, e.limits); err != nil {
			return err
		}
	}
	for _, rel := range mainRels {
		switch {
		case relIs(rel, relHeader), relIs(rel, relFooter):
			e.warnings.note(warnHeadersFooters)
		case relIs(rel, relComments):
			e.warnings.note(warnComments)
		}
	}

	err = r.readPart(main, mainRels, "document", func(c xml.StartElement) error {
		if c.Name.Local != "body" {
			return r.src.Skip()
		}
		return children(r.src, r.block)
	})
	if err != nil {
		return err
	}
	return r.readNotes(mainRels)
}

// readPart reads an XML part whose root has the local name rootName,
// handing each of the root's children to fn.
func (r *docxReader) readPart(part string, rels []officezip.Relationship, rootName string, fn func(xml.StartElement) error) error {
	dec, err := r.pkg.XML(part)
	if err != nil {
		return err
	}
	defer dec.Close()
	r.src, r.part, r.fields = dec, part, nil
	r.rels = make(map[string]officezip.Relationship, len(rels))
	for _, rel := range rels {
		r.rels[rel.ID] = rel
	}
	if _, err := root(dec, part, rootName); err != nil {
		return err
	}
	return children(dec, fn)
}

// readNotes writes the footnotes and then the endnotes the body referred
// to, under one "Notes" heading, in the order their parts hold them.
func (r *docxReader) readNotes(mainRels []officezip.Relationship) error {
	headed := false
	for _, kind := range []noteKind{footnote, endnote} {
		relType, element := relFootnotes, "footnote"
		if kind == endnote {
			relType, element = relEndnotes, "endnote"
		}
		marks := r.notes[kind].marks
		rel, ok := internalRel(mainRels, relType)
		if len(marks) == 0 || !ok || !r.pkg.Has(rel.Part) {
			continue
		}
		rels, err := r.pkg.Relationships(rel.Part)
		if err != nil {
			return err
		}
		err = r.readPart(rel.Part, rels, element+"s", func(c xml.StartElement) error {
			mark, referred := marks[attr(c, "id")]
			if typ := attr(c, "type"); c.Name.Local != element || !referred || (typ != "" && typ != "normal") {
				return r.src.Skip()
			}
			if !headed {
				headed = true
				if err := r.e.admit(); err != nil {
					return err
				}
				// The heading is Quark's, not the document's, so it is not
				// counted.
				if err := r.e.write(Block{Type: BlockHeading, Level: 1, Runs: []Run{{Text: "Notes"}}}); err != nil {
					return err
				}
			}
			r.note = mark
			defer func() { r.note = "" }()
			return children(r.src, r.block)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// block reads one block-level element.
func (r *docxReader) block(c xml.StartElement) error {
	switch c.Name.Local {
	case "p":
		return r.paragraph()
	case "tbl":
		return r.table()
	case "sdt":
		return r.sdt(r.block)
	case "customXml", "ins", "moveTo", "smartTag":
		return children(r.src, r.block)
	case "AlternateContent":
		return r.alternate(r.block)
	case "altChunk":
		if err := r.src.Skip(); err != nil {
			return xmlError(err)
		}
		return r.omit(nil, "embedded document")
	}
	// w:del and w:moveFrom are tracked deletions, read as accepted; the
	// rest (w:sectPr, bookmarks, proofing marks) has nothing to show.
	return r.src.Skip()
}

// sdt reads a content control's content with fn, and skips its properties.
func (r *docxReader) sdt(fn func(xml.StartElement) error) error {
	return children(r.src, func(c xml.StartElement) error {
		if c.Name.Local == "sdtContent" {
			return children(r.src, fn)
		}
		return r.src.Skip()
	})
}

// alternate reads the first choice of an mc:AlternateContent with fn, and
// skips the rest.
func (r *docxReader) alternate(fn func(xml.StartElement) error) error {
	chosen := false
	return children(r.src, func(c xml.StartElement) error {
		if (c.Name.Local == "Choice" || c.Name.Local == "Fallback") && !chosen {
			chosen = true
			return children(r.src, fn)
		}
		return r.src.Skip()
	})
}

// paragraph reads one w:p.
func (r *docxReader) paragraph() error {
	if err := r.e.admit(); err != nil {
		return err
	}
	if err := r.e.charge(blockOverhead); err != nil {
		return err
	}
	p := &paragraph{block: Block{Type: BlockParagraph}, admitted: true}
	ready := false
	err := children(r.src, func(c xml.StartElement) error {
		if c.Name.Local == "pPr" {
			props, err := readParaProps(r.src)
			if err != nil {
				return err
			}
			ready = true
			return r.setUp(p, props)
		}
		if !ready {
			ready = true
			if err := r.setUp(p, paraProps{outline: -1}); err != nil {
				return err
			}
		}
		return r.inline(p, c, "")
	})
	if !ready && err == nil {
		err = r.setUp(p, paraProps{outline: -1})
	}
	// A paragraph a limit cut short is still placed, with what it has; one
	// that failed is not.
	if err != nil && !errors.Is(err, errTruncated) {
		return err
	}
	if cerr := r.closeParagraph(p); err == nil {
		err = cerr
	}
	return err
}

// setUp gives a paragraph its type, alignment and base run formatting from
// its direct properties and its style.
func (r *docxReader) setUp(p *paragraph, direct paraProps) error {
	id := direct.style
	if id == "" {
		id = r.styles.defaultPara
	}
	style := r.styles.resolve(id)
	props := style.para.over(direct)
	p.base = r.styles.defaults.over(style.run)
	p.block.Align = props.align
	level := style.heading
	if direct.outline >= 0 {
		level = headingForOutline(direct.outline)
	}
	switch {
	case level > 0:
		p.block.Type, p.block.Level = BlockHeading, level
	case props.hasNum && props.numID != "" && props.numID != "0":
		if marker, ordered, ok := r.numbering.next(props.numID, props.ilvl); ok {
			p.block.Type, p.block.Level, p.block.Ordered, p.block.Marker = BlockListItem, props.ilvl, ordered, marker
		}
	}
	if direct.pageBreakBefore || style.para.pageBreakBefore {
		return r.pageBreak(p)
	}
	return nil
}

// inline reads one child of a paragraph, or of something inside one: link
// is the hyperlink it sits in, if any.
func (r *docxReader) inline(p *paragraph, c xml.StartElement, link string) error {
	within := func(link string) func(xml.StartElement) error {
		return func(c xml.StartElement) error { return r.inline(p, c, link) }
	}
	switch c.Name.Local {
	case "r":
		return r.run(p, link)
	case "hyperlink":
		if target := r.hyperlink(c); target != "" {
			link = target
		}
		return children(r.src, within(link))
	case "ins", "moveTo", "smartTag", "customXml", "fldSimple", "dir", "bdo":
		return children(r.src, within(link))
	case "sdt":
		return r.sdt(within(link))
	case "AlternateContent":
		return r.alternate(within(link))
	case "oMath", "oMathPara":
		if err := r.src.Skip(); err != nil {
			return xmlError(err)
		}
		return r.omit(p, "equation")
	}
	return r.src.Skip()
}

// run reads one w:r.
func (r *docxReader) run(p *paragraph, link string) error {
	props := p.base
	var child func(c xml.StartElement) error
	child = func(c xml.StartElement) error {
		switch c.Name.Local {
		case "rPr":
			direct, style, err := readRunProps(r.src)
			props = p.base
			if style != "" {
				props = props.over(r.styles.resolve(style).run)
			}
			props = props.over(direct)
			return err
		case "t":
			s, err := text(r.src)
			if err != nil {
				return err
			}
			return r.appendText(p, props, link, s)
		case "AlternateContent":
			return r.alternate(child)
		case "ruby":
			return children(r.src, func(c xml.StartElement) error {
				if c.Name.Local != "rubyBase" {
					return r.src.Skip()
				}
				return children(r.src, func(c xml.StartElement) error { return r.inline(p, c, link) })
			})
		case "drawing":
			info, err := scanDrawing(r.src)
			if err != nil {
				return err
			}
			return r.picture(p, info)
		case "pict":
			info, err := scanVML(r.src)
			if err != nil {
				return err
			}
			return r.picture(p, info)
		}

		// Everything else is an empty element, or one whose content is
		// never shown: w:instrText, w:delText, w:object.
		if err := r.src.Skip(); err != nil {
			return xmlError(err)
		}
		switch c.Name.Local {
		case "tab", "ptab":
			return r.appendText(p, props, link, "\t")
		case "br":
			if attr(c, "type") == "page" {
				return r.pageBreak(p)
			}
			return r.appendText(p, props, link, "\n")
		case "cr":
			return r.appendText(p, props, link, "\n")
		case "noBreakHyphen":
			return r.appendText(p, props, link, "-")
		case "sym":
			return r.appendText(p, props, link, symbol(attr(c, "char")))
		case "fldChar":
			r.field(attr(c, "fldCharType"))
		case "object":
			return r.omit(p, "embedded object")
		case "footnoteReference":
			return r.noteRef(p, props, footnote, attr(c, "id"))
		case "endnoteReference":
			return r.noteRef(p, props, endnote, attr(c, "id"))
		case "footnoteRef", "endnoteRef":
			return r.appendText(p, superscript(props), "", r.note)
		}
		return nil
	}
	return children(r.src, child)
}

// field follows a complex field's w:fldChar: its instructions run from
// begin to separate, its cached result from separate to end.
func (r *docxReader) field(kind string) {
	switch kind {
	case "begin":
		r.fields = append(r.fields, false)
	case "separate":
		if n := len(r.fields); n > 0 {
			r.fields[n-1] = true
		}
	case "end":
		if n := len(r.fields); n > 0 {
			r.fields = r.fields[:n-1]
		}
	}
	if len(r.fields) > maxFieldDepth {
		// Past the bound every field is instructions, and stays so until
		// enough of them end.
		r.fields = r.fields[:maxFieldDepth]
		r.fields[maxFieldDepth-1] = false
	}
}

// inFieldCode reports that text is inside a field's instructions, which
// are never shown.
func (r *docxReader) inFieldCode() bool {
	for _, result := range r.fields {
		if !result {
			return true
		}
	}
	return false
}

// appendText adds text to the paragraph with the run's formatting. Hidden
// text and field instructions are dropped.
func (r *docxReader) appendText(p *paragraph, props runProps, link, s string) error {
	if props.hidden == optOn || r.inFieldCode() {
		return nil
	}
	return r.addText(p, props.run("", link), s)
}

// picture places what a drawing turned out to be: an image from the
// package, a linked image that is never fetched, or an omitted block.
func (r *docxReader) picture(p *paragraph, info drawingInfo) error {
	if info.what != "" {
		return r.omit(p, info.what)
	}
	if info.link != "" {
		return r.linkedImage(p, info)
	}
	rel, ok := r.rels[info.embed]
	switch {
	case !ok:
		return r.omit(p, "image")
	case rel.External:
		return r.linkedImage(p, info)
	case rel.Part == "" || !r.pkg.Has(rel.Part):
		return r.omit(p, "image")
	}
	return r.split(p, Block{Type: BlockImage, Part: rel.Part, Alt: info.alt, Width: info.width, Height: info.height})
}

// hyperlink returns a w:hyperlink's target, when it is one a reader may
// open: an external http, https or mailto link. A link to a place inside
// the document has no target here.
func (r *docxReader) hyperlink(c xml.StartElement) string {
	rel, ok := r.rels[attr(c, "id")]
	if !ok || !rel.External {
		return ""
	}
	return safeLink(rel.Target)
}

// safeLink returns target if it is an http, https or mailto URL, and empty
// otherwise.
func safeLink(target string) string {
	target = strings.TrimSpace(target)
	if len(target) > maxLinkLength {
		return ""
	}
	u, err := url.Parse(target)
	if err != nil {
		return ""
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		if u.Host == "" {
			return ""
		}
		return target
	case "mailto":
		return target
	}
	return ""
}

// noteRef shows a note's mark where the body refers to it, numbering the
// note the first time.
func (r *docxReader) noteRef(p *paragraph, props runProps, kind noteKind, id string) error {
	m := &r.notes[kind]
	mark, ok := m.marks[id]
	if !ok {
		if len(m.marks) >= r.e.limits.MaxDefinitions {
			r.e.warnings.note(warnNotes)
			return nil
		}
		if m.marks == nil {
			m.marks = make(map[string]string)
		}
		m.next++
		mark = strconv.Itoa(m.next)
		if kind == endnote {
			mark = roman(min(m.next, 3999))
		}
		m.marks[id] = mark
	}
	return r.appendText(p, superscript(props), "", mark)
}

// superscript is props raised as a note mark.
func superscript(props runProps) runProps {
	props.sup, props.sub = optOn, optOff
	return props
}

// symbol is the character a w:sym names, or nothing when it names one of a
// symbol font's own, which no other font draws.
func symbol(hex string) string {
	n, err := strconv.ParseUint(hex, 16, 32)
	if err != nil || n == 0 || (n >= 0xE000 && n <= 0xF8FF) || n > 0x10FFFF || (n >= 0xD800 && n <= 0xDFFF) {
		return ""
	}
	return string(rune(n))
}

// table reads one w:tbl. A table nested past the depth limit is flattened
// into the cell it sits in.
func (r *docxReader) table() error {
	if r.tableDepth >= r.e.limits.MaxTableDepth {
		return r.flattenTable()
	}
	if err := r.e.admit(); err != nil {
		return err
	}
	if err := r.e.charge(blockOverhead); err != nil {
		return err
	}
	t := &tableBuilder{merging: map[int]cellIndex{}}
	outer := r.cell
	r.tableDepth++
	var child func(c xml.StartElement) error
	child = func(c xml.StartElement) error {
		switch c.Name.Local {
		case "tr":
			return r.row(t)
		case "sdt":
			return r.sdt(child)
		case "customXml", "ins", "moveTo":
			return children(r.src, child)
		}
		return r.src.Skip()
	}
	err := children(r.src, child)
	r.tableDepth--
	r.cell = outer
	// A table a limit cut short is still placed, with the rows it has; one
	// that failed is not.
	if err != nil && !errors.Is(err, errTruncated) {
		return err
	}
	if placeErr := r.put(Block{Type: BlockTable, Rows: t.rows}); err == nil {
		err = placeErr
	}
	return err
}

// row reads one w:tr. A row deleted as a tracked change is dropped.
func (r *docxReader) row(t *tableBuilder) error {
	if err := r.e.charge(rowOverhead); err != nil {
		return err
	}
	t.rows = append(t.rows, TableRow{Cells: []TableCell{}})
	at := len(t.rows) - 1
	col, deleted := 0, false
	var child func(c xml.StartElement) error
	child = func(c xml.StartElement) error {
		if deleted {
			return r.src.Skip()
		}
		switch c.Name.Local {
		case "trPr":
			return children(r.src, func(c xml.StartElement) error {
				switch c.Name.Local {
				case "del":
					deleted = true
				case "gridBefore":
					col += clampAtoi(attr(c, "val"), 0, maxSpan)
				}
				return r.src.Skip()
			})
		case "tc":
			return r.tableCell(t, at, &col)
		case "sdt":
			return r.sdt(child)
		case "customXml", "ins", "moveTo":
			return children(r.src, child)
		}
		return r.src.Skip()
	}
	err := children(r.src, child)
	if deleted {
		t.rows = t.rows[:at]
	}
	return err
}

// maxSpan bounds a cell's column span; Word's own limit is 63 columns.
const maxSpan = 64

// tableCell reads one w:tc of row at, whose grid column col points at.
func (r *docxReader) tableCell(t *tableBuilder, at int, col *int) error {
	if err := r.e.charge(cellOverhead); err != nil {
		return err
	}
	cell := TableCell{ColSpan: 1, RowSpan: 1, Blocks: []Block{}}
	merge := ""
	outer := r.cell
	r.cell = &cell.Blocks
	err := children(r.src, func(c xml.StartElement) error {
		if c.Name.Local != "tcPr" {
			return r.block(c)
		}
		return children(r.src, func(c xml.StartElement) error {
			switch c.Name.Local {
			case "gridSpan":
				cell.ColSpan = clampAtoi(attr(c, "val"), 1, maxSpan)
			case "vMerge":
				merge = "continue"
				if attr(c, "val") == "restart" {
					merge = "restart"
				}
			}
			return r.src.Skip()
		})
	})
	r.cell = outer

	start := *col
	*col += cell.ColSpan
	if origin, ok := t.merging[start]; ok && merge == "continue" {
		// A merged-away cell lengthens the cell the merge started in, which
		// takes whatever it holds.
		o := &t.rows[origin.row].Cells[origin.cell]
		o.RowSpan++
		for _, b := range cell.Blocks {
			if b.Type != BlockParagraph || len(b.Runs) > 0 {
				o.Blocks = append(o.Blocks, b)
			}
		}
		return err
	}
	row := &t.rows[at]
	row.Cells = append(row.Cells, cell)
	if merge == "restart" {
		t.merging[start] = cellIndex{row: at, cell: len(row.Cells) - 1}
	} else {
		delete(t.merging, start)
	}
	return err
}

// flattenTable reads a table nested too deeply as the blocks of its cells,
// one after another.
func (r *docxReader) flattenTable() error {
	return children(r.src, func(c xml.StartElement) error {
		if c.Name.Local != "tr" {
			return r.src.Skip()
		}
		return children(r.src, func(c xml.StartElement) error {
			if c.Name.Local != "tc" {
				return r.src.Skip()
			}
			return children(r.src, func(c xml.StartElement) error {
				if c.Name.Local == "tcPr" {
					return r.src.Skip()
				}
				return r.block(c)
			})
		})
	})
}

// internalRel finds the first relationship of a type with a target inside
// the package.
func internalRel(rels []officezip.Relationship, relType string) (officezip.Relationship, bool) {
	for _, rel := range rels {
		if relIs(rel, relType) && !rel.External && rel.Part != "" {
			return rel, true
		}
	}
	return officezip.Relationship{}, false
}

// relIs reports whether a relationship has the type whose last segment is
// suffix.
func relIs(rel officezip.Relationship, suffix string) bool {
	return strings.HasSuffix(rel.Type, suffix)
}
