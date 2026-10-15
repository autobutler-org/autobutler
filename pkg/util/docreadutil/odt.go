package docreadutil

// cspell:ignore applet

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"slices"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/officezip"
)

// The parts of an ODT package the reader opens.
const (
	odtContentPart = "content.xml"
	odtStylesPart  = "styles.xml"
	odtMetaPart    = "meta.xml"
)

// maxRepeat bounds how many times a table row or cell repeats; a writer
// repeats empty ones, and every repeat is charged as if written out.
const maxRepeat = 1000

// maxRowSpan bounds a cell's row span before it is cut to the rows the
// table has.
const maxRowSpan = 1 << 16

// maxSpaces bounds the spaces one text:s stands for.
const maxSpaces = 1000

// odtShapes are the drawing elements shown as an omitted shape.
var odtShapes = map[string]bool{
	"custom-shape": true, "rect": true, "line": true, "polyline": true, "polygon": true,
	"regular-polygon": true, "path": true, "circle": true, "ellipse": true, "connector": true,
	"caption": true, "measure": true, "g": true, "control": true,
}

// readOdt reads an ODT package's body, then the notes it cited.
func readOdt(e *emitter, src io.ReaderAt, size int64) error {
	pkg, err := officezip.Open(src, size, e.limits.Zip)
	if err != nil {
		return err
	}
	if !pkg.Has(odtContentPart) {
		return fmt.Errorf("%w: the package holds no text document", ErrMalformed)
	}
	r := &odtReader{
		blockPlacer: blockPlacer{e: e},
		pkg:         pkg,
		styles:      newOdtStyles(e.limits),
		objects:     map[string]string{},
		byStyle:     map[string]*odtCounter{},
		byID:        map[string]*odtCounter{},
		notes:       map[string]string{},
	}
	if err := r.readManifest(); err != nil {
		return err
	}
	if pkg.Has(odtMetaPart) {
		e.title = readTitle(pkg, odtMetaPart, e.limits.MaxHeldBytes, "document-meta", "meta")
	}
	if pkg.Has(odtStylesPart) {
		if err := r.readStylesPart(); err != nil {
			return err
		}
	}
	if err := r.readContent(); err != nil {
		return err
	}
	return r.readNotes()
}

// readManifest notes the media type of each embedded object, and refuses a
// document whose content is encrypted. A package without a manifest is
// read all the same.
func (r *odtReader) readManifest() error {
	entries, err := r.pkg.Manifest()
	if errors.Is(err, officezip.ErrMissingPart) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := strings.TrimSuffix(strings.TrimPrefix(entry.FullPath, "./"), "/")
		if entry.Encrypted && name == odtContentPart {
			return fmt.Errorf("%w: the document is protected by a password", ErrMalformed)
		}
		if strings.HasSuffix(entry.FullPath, "/") && name != "" {
			r.objects[name] = entry.MediaType
		}
	}
	return nil
}

// readStylesPart reads and holds styles.xml: its font faces, styles and
// list styles, and whether its page styles have headers or footers.
func (r *odtReader) readStylesPart() error {
	dec, closer, err := heldXML(r.pkg, odtStylesPart, r.e.limits.MaxHeldBytes)
	if err != nil {
		return err
	}
	defer closer.Close()
	if _, err := root(dec, odtStylesPart, "document-styles"); err != nil {
		return err
	}
	return children(dec, func(c xml.StartElement) error {
		switch c.Name.Local {
		case "font-face-decls":
			return r.styles.readFontFaces(dec)
		case "styles":
			return r.styles.readStyles(dec, false)
		case "automatic-styles":
			// styles.xml's automatic styles serve its headers and footers,
			// never the body.
			return dec.Skip()
		case "master-styles":
			return scan(dec, func(c xml.StartElement) {
				if strings.HasPrefix(c.Name.Local, "header") || strings.HasPrefix(c.Name.Local, "footer") {
					r.e.warnings.note(warnHeadersFooters)
				}
			})
		}
		return dec.Skip()
	})
}

// readContent streams content.xml: its automatic styles, then its body.
func (r *odtReader) readContent() error {
	dec, err := r.pkg.XML(odtContentPart)
	if err != nil {
		return err
	}
	defer dec.Close()
	r.src = dec
	if _, err := root(dec, odtContentPart, "document-content"); err != nil {
		return err
	}
	sawText := false
	err = children(dec, func(c xml.StartElement) error {
		switch c.Name.Local {
		case "font-face-decls":
			return r.styles.readFontFaces(dec)
		case "automatic-styles":
			return r.styles.readStyles(dec, true)
		case "body":
			return children(dec, func(c xml.StartElement) error {
				if c.Name.Local != "text" {
					return dec.Skip()
				}
				sawText = true
				return children(dec, r.block)
			})
		}
		return dec.Skip()
	})
	if err == nil && !sawText {
		return fmt.Errorf("%w: the package is not a text document", ErrMalformed)
	}
	return err
}

// readNotes reads content.xml a second time for the bodies of the notes
// the first pass cited, and writes them under one "Notes" heading, in the
// order the document holds them. Reading the part twice keeps a note's
// body from being held until the end of the document.
func (r *odtReader) readNotes() error {
	if len(r.notes) == 0 {
		return nil
	}
	dec, err := r.pkg.XML(odtContentPart)
	if err != nil {
		return err
	}
	defer dec.Close()
	r.src = dec
	if _, err := root(dec, odtContentPart, "document-content"); err != nil {
		return err
	}
	headed := false
	var find func(c xml.StartElement) error
	find = func(c xml.StartElement) error {
		switch c.Name.Local {
		case "automatic-styles", "font-face-decls", "tracked-changes", "annotation":
			return dec.Skip()
		case "note":
		default:
			return children(dec, find)
		}
		mark, cited := r.notes[attr(c, "id")]
		if !cited {
			return dec.Skip()
		}
		delete(r.notes, attr(c, "id"))
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
		return children(dec, func(c xml.StartElement) error {
			if c.Name.Local != "note-body" {
				return dec.Skip()
			}
			r.note = mark
			defer func() { r.note = "" }()
			return children(dec, r.block)
		})
	}
	return children(dec, find)
}

// block reads one block-level element.
func (r *odtReader) block(c xml.StartElement) error {
	switch c.Name.Local {
	case "p":
		return r.paragraph(c, false)
	case "h":
		return r.paragraph(c, true)
	case "list":
		return r.list(c)
	case "table":
		return r.table()
	case "section":
		if attr(c, "display") == "none" {
			return r.src.Skip()
		}
		return children(r.src, r.block)
	case "numbered-paragraph", "index-body", "index-title", "table-of-content", "alphabetical-index",
		"illustration-index", "table-index", "object-index", "user-index", "bibliography":
		// An index's generated text is in its index-body; its
		// index-source, the template it was generated from, is skipped.
		return children(r.src, r.block)
	case "frame":
		return r.frame(nil, c)
	}
	if odtShapes[c.Name.Local] {
		if err := r.src.Skip(); err != nil {
			return xmlError(err)
		}
		return r.omit(nil, "shape")
	}
	// Tracked changes are read as accepted: an insertion is in the body
	// already, and text:tracked-changes holds only what was deleted.
	// Declarations, forms and soft page breaks have nothing to show.
	return r.src.Skip()
}

// paragraph reads one text:p, or a text:h when heading is set.
func (r *odtReader) paragraph(c xml.StartElement, heading bool) error {
	if err := r.e.admit(); err != nil {
		return err
	}
	if err := r.e.charge(blockOverhead); err != nil {
		return err
	}
	style := r.styles.paragraph(attr(c, "style-name"))
	p := &paragraph{block: Block{Type: BlockParagraph, Align: style.align}, base: style.run, admitted: true}
	item := r.item
	r.item = nil
	switch {
	case heading:
		level := clampAtoi(attr(c, "outline-level"), 0, 10)
		if level == 0 {
			level = max(style.outline, 1)
		}
		p.block.Type, p.block.Level = BlockHeading, min(level, 6)
	case style.heading > 0:
		p.block.Type, p.block.Level = BlockHeading, style.heading
	case item != nil:
		lsName := r.listStyle
		if lsName == "" {
			lsName = style.listStyle
		}
		level := min(r.listDepth-1, odtLevels-1)
		p.block.Marker, p.block.Ordered = r.counter.marker(r.styles.lists[lsName], level, item.start)
		p.block.Type, p.block.Level = BlockListItem, level
	}
	r.space = true
	var err error
	if style.breakBefore {
		err = r.pageBreak(p)
	}
	if r.note != "" && err == nil {
		// The mark, then a space, which the white space a note's text
		// starts with collapses into.
		err = r.addText(p, superscript(p.base).run("", ""), r.note)
		if err == nil {
			err = r.addRun(p, p.base, "", " ")
		}
		r.note = ""
	}
	if err == nil {
		err = r.mixed(p, p.base, "")
	}
	// A paragraph a limit cut short is still placed, with what it has; one
	// that failed is not.
	if err != nil && !errors.Is(err, errTruncated) {
		return err
	}
	if cerr := r.closeParagraph(p); err == nil {
		err = cerr
	}
	if style.breakAfter && err == nil {
		err = r.pageBreak(nil)
	}
	return err
}

// mixed reads the mixed content of the element whose start was just read,
// through to its end: text with props and link, and inline elements.
func (r *odtReader) mixed(p *paragraph, props runProps, link string) error {
	for {
		tok, err := r.src.Token()
		if err != nil {
			return xmlError(err)
		}
		switch t := tok.(type) {
		case xml.CharData:
			if err := r.addRun(p, props, link, r.collapse(t)); err != nil {
				return err
			}
		case xml.StartElement:
			if err := r.inline(p, t, props, link); err != nil {
				return err
			}
		case xml.EndElement:
			return nil
		}
	}
}

// collapse folds white space as ODF reads it: every run of spaces, tabs
// and line ends is one space, and white space at the start of a paragraph
// is none. Spaces that count are written as text:s.
func (r *odtReader) collapse(data []byte) string {
	var sb strings.Builder
	for _, b := range data {
		if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
			if !r.space {
				sb.WriteByte(' ')
			}
			r.space = true
			continue
		}
		sb.WriteByte(b)
		r.space = false
	}
	return sb.String()
}

// addRun adds text to the paragraph unless its formatting hides it.
func (r *odtReader) addRun(p *paragraph, props runProps, link, s string) error {
	if props.hidden == optOn {
		return nil
	}
	return r.addText(p, props.run("", link), s)
}

// inline reads one element inside a paragraph.
func (r *odtReader) inline(p *paragraph, c xml.StartElement, props runProps, link string) error {
	switch c.Name.Local {
	case "span":
		return r.mixed(p, props.over(r.styles.text(attr(c, "style-name"))), link)
	case "a":
		if target := safeLink(attr(c, "href")); target != "" {
			link = target
		}
		return r.mixed(p, props, link)
	case "note":
		return r.noteCitation(p, c, props)
	case "frame":
		return r.frame(p, c)
	case "annotation":
		r.e.warnings.note(warnComments)
		return r.src.Skip()
	case "s", "tab", "line-break":
		if err := r.src.Skip(); err != nil {
			return xmlError(err)
		}
		s := "\t"
		switch c.Name.Local {
		case "s":
			s = strings.Repeat(" ", clampAtoi(attr(c, "c"), 1, maxSpaces))
		case "line-break":
			s = "\n"
		}
		r.space = c.Name.Local != "s"
		return r.addRun(p, props, link, s)
	case "number", "ruby-text", "script", "hidden-text", "hidden-paragraph", "event-listeners",
		"tracked-changes", "title", "desc":
		// A list number the writer cached, which the reader numbers
		// itself; ruby annotations; and what is never shown.
		return r.src.Skip()
	}
	if odtShapes[c.Name.Local] {
		if err := r.src.Skip(); err != nil {
			return xmlError(err)
		}
		return r.omit(p, "shape")
	}
	// A field shows the result its writer cached; a ruby its base text;
	// marks, bookmarks and change marks hold nothing.
	return r.mixed(p, props, link)
}

// noteCitation shows a note's mark where the body cites it, and remembers
// the note so its body follows the document. A note with no id cannot be
// found again, and one past the definitions limit is not looked for.
func (r *odtReader) noteCitation(p *paragraph, c xml.StartElement, props runProps) error {
	id, mark := attr(c, "id"), ""
	err := children(r.src, func(c xml.StartElement) error {
		if c.Name.Local != "note-citation" {
			return r.src.Skip()
		}
		mark = attr(c, "label")
		t, err := text(r.src)
		if mark == "" {
			mark = strings.TrimSpace(t)
		}
		return err
	})
	if err != nil {
		return err
	}
	if _, dup := r.notes[id]; id == "" || dup || len(r.notes) >= r.e.limits.MaxDefinitions {
		r.e.warnings.note(warnNotes)
	} else {
		r.notes[id] = mark
	}
	r.space = false
	return r.addRun(p, superscript(props), "", mark)
}

// frame reads a draw:frame: an image from the package, a linked image that
// is never fetched, a text box's blocks, or an omitted object.
func (r *odtReader) frame(p *paragraph, c xml.StartElement) error {
	width, height := cssPixels(attr(c, "width")), cssPixels(attr(c, "height"))
	var what, href, title, desc string
	image, inline, boxed := false, false, false
	err := children(r.src, func(c xml.StartElement) error {
		switch c.Name.Local {
		case "image":
			if !image {
				image, href = true, attr(c, "href")
				return scan(r.src, func(c xml.StartElement) { inline = inline || c.Name.Local == "binary-data" })
			}
		case "object":
			what = firstNonEmpty(what, r.objectKind(attr(c, "href")))
		case "object-ole", "plugin", "applet":
			what = firstNonEmpty(what, "embedded object")
		case "floating-frame":
			what = firstNonEmpty(what, "embedded document")
		case "text-box":
			if what == "" && !boxed {
				boxed = true
				return r.textBox(p)
			}
		case "title", "desc":
			t, err := text(r.src)
			if c.Name.Local == "title" {
				title = t
			} else {
				desc = t
			}
			return err
		}
		return r.src.Skip()
	})
	if err != nil {
		return err
	}
	// An object's draw:image is the picture of it its writer cached, never
	// shown in its place.
	info := drawingInfo{alt: firstNonEmpty(desc, title), width: width, height: height}
	switch {
	case what != "":
		return r.omit(p, what)
	case boxed || !image:
		return nil
	case inline || href == "":
		return r.omit(p, "image")
	}
	part, internal := packagePath(href)
	switch {
	case !internal:
		return r.linkedImage(p, info)
	case !r.pkg.Has(part):
		return r.omit(p, "image")
	}
	return r.split(p, Block{Type: BlockImage, Part: part, Alt: info.alt, Width: width, Height: height})
}

// textBox reads a draw:text-box's blocks where the frame sits, splitting
// the paragraph p it interrupts, if any. A captioned picture is a text box
// holding the picture and its caption.
func (r *odtReader) textBox(p *paragraph) error {
	if p != nil {
		if err := r.flush(p); err != nil {
			return err
		}
		p.placed = true
	}
	err := children(r.src, r.block)
	r.space = true
	return err
}

// objectKind names what an embedded object is from the media type the
// manifest gives its directory.
func (r *odtReader) objectKind(href string) string {
	name, _ := packagePath(href)
	switch mediaType := r.objects[name]; {
	case strings.HasSuffix(mediaType, ".chart"):
		return "chart"
	case strings.HasSuffix(mediaType, ".formula"):
		return "equation"
	}
	return "embedded object"
}

// packagePath resolves an xlink:href to a part name, and reports whether it
// names one inside the package: a URL, an absolute path and one that
// climbs out of the package do not.
func packagePath(href string) (string, bool) {
	href = strings.TrimSpace(href)
	if u, err := url.Parse(href); err != nil || u.Scheme != "" || u.Host != "" || strings.HasPrefix(href, "/") {
		return "", false
	}
	name := path.Clean(href)
	if name == "." || name == ".." || strings.HasPrefix(name, "../") {
		return "", false
	}
	return name, true
}

// list reads one text:list. A list inherits the style of the list it sits
// in; an outermost one starts its numbering again unless it continues an
// earlier list's.
func (r *odtReader) list(c xml.StartElement) error {
	style := attr(c, "style-name")
	if style == "" {
		style = r.listStyle
	}
	savedStyle, savedCounter := r.listStyle, r.counter
	if r.listDepth == 0 {
		counter := &odtCounter{style: style}
		if prior, ok := r.byID[attr(c, "continue-list")]; ok {
			counter = prior
		} else if prior, ok := r.byStyle[style]; ok && attr(c, "continue-numbering") == "true" {
			counter = prior
		}
		if style == "" {
			style = counter.style
		}
		r.remember(r.byStyle, style, counter)
		if id := attr(c, "id"); id != "" {
			r.remember(r.byID, id, counter)
		}
		r.counter = counter
	}
	r.listDepth++
	r.listStyle, r.item = style, nil
	err := children(r.src, func(c xml.StartElement) error {
		switch c.Name.Local {
		case "list-item":
			r.item = &odtItem{start: attr(c, "start-value")}
		case "list-header":
			r.item = nil
		default:
			return r.src.Skip()
		}
		err := children(r.src, r.block)
		r.item = nil
		return err
	})
	r.listDepth--
	r.listStyle, r.counter, r.item = savedStyle, savedCounter, nil
	return err
}

// remember records a list's numbering under a key, while there is room.
func (r *odtReader) remember(m map[string]*odtCounter, key string, counter *odtCounter) {
	if _, ok := m[key]; ok || len(m) < r.e.limits.MaxDefinitions {
		m[key] = counter
	}
}

// table reads one table:table. A table nested past the depth limit is
// flattened into the cell it sits in.
func (r *odtReader) table() error {
	if r.tableDepth >= r.e.limits.MaxTableDepth {
		return r.flattenTable()
	}
	if err := r.e.admit(); err != nil {
		return err
	}
	if err := r.e.charge(blockOverhead); err != nil {
		return err
	}
	var rows []TableRow
	outer, item := r.cell, r.item
	r.tableDepth++
	r.item = nil
	var child func(c xml.StartElement) error
	child = func(c xml.StartElement) error {
		switch c.Name.Local {
		case "table-row":
			return r.row(&rows, c)
		case "table-header-rows", "table-rows", "table-row-group":
			return children(r.src, child)
		}
		return r.src.Skip()
	}
	err := children(r.src, child)
	r.tableDepth--
	r.cell, r.item = outer, item
	// A table a limit cut short is still placed, with the rows it has; one
	// that failed is not.
	if err != nil && !errors.Is(err, errTruncated) {
		return err
	}
	// A span past the last row ends with the table.
	for i := range rows {
		for j := range rows[i].Cells {
			rows[i].Cells[j].RowSpan = min(rows[i].Cells[j].RowSpan, len(rows)-i)
		}
	}
	if placeErr := r.put(Block{Type: BlockTable, Rows: rows}); err == nil {
		err = placeErr
	}
	return err
}

// row reads one table:table-row, repeated as many times as it says. A cell
// a span covers is not listed.
func (r *odtReader) row(rows *[]TableRow, c xml.StartElement) error {
	held := r.e.held
	if err := r.e.charge(rowOverhead); err != nil {
		return err
	}
	row := TableRow{Cells: []TableCell{}}
	err := children(r.src, func(c xml.StartElement) error {
		if c.Name.Local == "table-cell" {
			return r.tableCell(&row, c)
		}
		return r.src.Skip()
	})
	*rows = append(*rows, row)
	if err != nil {
		return err
	}
	return r.repeat(attr(c, "number-rows-repeated"), held, func() {
		*rows = append(*rows, TableRow{Cells: slices.Clone(row.Cells)})
	})
}

// tableCell reads one table:table-cell, repeated as many times as it says.
func (r *odtReader) tableCell(row *TableRow, c xml.StartElement) error {
	held := r.e.held
	if err := r.e.charge(cellOverhead); err != nil {
		return err
	}
	cell := TableCell{
		ColSpan: clampAtoi(attr(c, "number-columns-spanned"), 1, maxSpan),
		RowSpan: clampAtoi(attr(c, "number-rows-spanned"), 1, maxRowSpan),
		Blocks:  []Block{},
	}
	outer := r.cell
	r.cell = &cell.Blocks
	err := children(r.src, r.block)
	r.cell = outer
	row.Cells = append(row.Cells, cell)
	if err != nil {
		return err
	}
	return r.repeat(attr(c, "number-columns-repeated"), held, func() {
		row.Cells = append(row.Cells, cell)
	})
}

// repeat adds the repeats of a row or cell that a repeat count asks for
// beyond the first, charging each what the first cost: what is held now
// above held.
func (r *odtReader) repeat(count string, held int64, add func()) error {
	cost := int(r.e.held - held)
	for range clampAtoi(count, 1, maxRepeat) - 1 {
		if err := r.e.charge(cost); err != nil {
			return err
		}
		add()
	}
	return nil
}

// flattenTable reads a table nested too deeply as the blocks of its cells,
// one after another.
func (r *odtReader) flattenTable() error {
	var child func(c xml.StartElement) error
	child = func(c xml.StartElement) error {
		switch c.Name.Local {
		case "table-header-rows", "table-rows", "table-row-group", "table-row":
			return children(r.src, child)
		case "table-cell":
			return children(r.src, r.block)
		}
		return r.src.Skip()
	}
	return children(r.src, child)
}
