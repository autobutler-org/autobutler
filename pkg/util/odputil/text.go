package odputil

import (
	"fmt"
	"strconv"
	"strings"
)

// paragraphs reads the text:p, text:h and text:list children of container —
// a text box, a shape, a table cell — drawn with frameRefs under their own
// styles.
func (s *slideReader) paragraphs(container *node, frameRefs []styleRef) ([]outParagraph, error) {
	var out []outParagraph
	for _, c := range container.children {
		var err error
		switch {
		case c.is(nsText, "p"), c.is(nsText, "h"):
			var p outParagraph
			p, err = s.paragraph(c, frameRefs, "")
			out = append(out, p)
		case c.is(nsText, "list"):
			out, err = s.list(c, frameRefs, "", 1, out)
		}
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// list appends a text:list's paragraphs, bulleted or numbered as its list
// style says for level; a nested list is the next level, and a list without
// a style of its own numbers with its parent's.
func (s *slideReader) list(n *node, frameRefs []styleRef, name string, level int, out []outParagraph) ([]outParagraph, error) {
	if own := n.attr(nsText, "style-name"); own != "" {
		name = own
	}
	kind := s.styles.listKind(s.scope, name, level)
	for _, item := range n.children {
		header := item.is(nsText, "list-header")
		if !header && !item.is(nsText, "list-item") {
			continue
		}
		for _, c := range item.children {
			var err error
			switch {
			case c.is(nsText, "p"), c.is(nsText, "h"):
				itemKind := kind
				if header {
					itemKind = ""
				}
				var p outParagraph
				p, err = s.paragraph(c, frameRefs, itemKind)
				out = append(out, p)
			case c.is(nsText, "list"):
				out, err = s.list(c, frameRefs, name, level+1, out)
			}
			if err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

// paragraph converts one text:p or text:h: its runs, alignment and line
// spacing, and list when it is a list item.
func (s *slideReader) paragraph(p *node, frameRefs []styleRef, list string) (outParagraph, error) {
	refs := append([]styleRef{{scope: s.scope, family: "paragraph", name: p.attr(nsText, "style-name")}}, frameRefs...)
	if s.styles.get(refs, "paragraph-properties", "text:enable-numbering") == "false" {
		list = ""
	}
	w := &textWalker{s: s}
	w.walk(p, refs)
	w.end()
	if w.err != nil {
		return outParagraph{}, w.err
	}
	out := outParagraph{Runs: w.runs, List: list}
	if out.Runs == nil {
		out.Runs = []outRun{}
	}
	switch s.styles.get(refs, "paragraph-properties", "fo:text-align") {
	case "center":
		out.Align = "center"
	case "end", "right":
		out.Align = "end"
	case "justify":
		out.Align = "justify"
	}
	if v, ok := parsePercent(s.styles.get(refs, "paragraph-properties", "fo:line-height")); ok && v > 0 {
		spacing := round(v)
		out.LineSpacing = &spacing
	}
	return out, nil
}

// walk adds the runs of n's children: text, spans, spaces, tabs, line breaks,
// links and fields, whose shown text is what they hold.
func (w *textWalker) walk(n *node, refs []styleRef) {
	for _, c := range n.children {
		if w.err != nil {
			return
		}
		if c.name.Local == "" {
			w.text(c.text, refs)
			continue
		}
		if c.name.Space != nsText {
			continue
		}
		switch c.name.Local {
		case "span":
			span := append([]styleRef{{scope: w.s.scope, family: "text", name: c.attr(nsText, "style-name")}}, refs...)
			w.walk(c, span)
		case "s":
			count := min(max(atoiDefault(c.attr(nsText, "c"), 1), 1), maxSpaceRun)
			w.literal(strings.Repeat(" ", count), refs)
		case "tab":
			w.literal("\t", refs)
		case "line-break":
			w.literal("\n", refs)
			// White space after a line break starts the next line.
			w.started, w.lastSpace = false, true
		case "note", "bookmark", "bookmark-start", "bookmark-end", "soft-page-break":
		default:
			w.walk(c, refs)
		}
	}
}

// text adds file text, collapsing its white space: the first of a run is
// kept as a space, the rest dropped.
func (w *textWalker) text(t string, refs []styleRef) {
	var b strings.Builder
	for _, r := range t {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if w.started && !w.lastSpace {
				b.WriteByte(' ')
			}
			w.lastSpace = true
			continue
		}
		w.started, w.lastSpace = true, false
		b.WriteRune(r)
	}
	w.emit(b.String(), refs)
}

// literal adds text the file spells out, which is never collapsed.
func (w *textWalker) literal(t string, refs []styleRef) {
	w.started, w.lastSpace = true, false
	w.emit(t, refs)
}

// end finishes the last run, without the space a paragraph's trailing
// white space left.
func (w *textWalker) end() {
	w.flush()
	if !w.lastSpace || !w.started || len(w.runs) == 0 {
		return
	}
	last := &w.runs[len(w.runs)-1]
	last.Text = strings.TrimSuffix(last.Text, " ")
	if last.Text == "" {
		w.runs = w.runs[:len(w.runs)-1]
	}
}

// emit appends t in refs' style, to the last run when it is styled the same.
func (w *textWalker) emit(t string, refs []styleRef) {
	if t == "" {
		return
	}
	w.s.textBytes += len(t)
	if w.s.textBytes > maxSlideTextBytes {
		w.err = fmt.Errorf("%w: a slide holds more than %d bytes of text", ErrTooLarge, maxSlideTextBytes)
		return
	}
	run := w.s.runStyle(refs)
	if len(w.runs) == 0 || !sameStyle(w.runs[len(w.runs)-1], run) {
		w.flush()
		w.runs = append(w.runs, run)
	}
	w.cur.WriteString(t)
}

// flush sets the last run's text from what has been written to it.
func (w *textWalker) flush() {
	if len(w.runs) > 0 && w.cur.Len() > 0 {
		w.runs[len(w.runs)-1].Text = w.cur.String()
	}
	w.cur.Reset()
}

// runStyle is the style text drawn with refs takes: what the nearest style
// sets of each property.
func (s *slideReader) runStyle(refs []styleRef) outRun {
	get := func(key string) string { return s.styles.get(refs, "text-properties", key) }
	var run outRun
	switch weight := get("fo:font-weight"); weight {
	case "bold":
		run.Bold = true
	default:
		if n, err := strconv.Atoi(weight); err == nil && n >= 600 {
			run.Bold = true
		}
	}
	style := get("fo:font-style")
	run.Italic = style == "italic" || style == "oblique"
	if u := get("style:text-underline-style"); u != "" && u != "none" {
		run.Underline = true
	}
	if l := get("style:text-line-through-style"); l != "" && l != "none" {
		run.Strikethrough = true
	}
	if size, ok := s.length(get("fo:font-size")); ok && size > 0 {
		v := round(size)
		run.FontSize = &v
	}
	family := s.styles.fonts[get("style:font-name")]
	if family == "" {
		family = fontFamily(get("fo:font-family"))
	}
	if family != "" {
		run.FontFamily = &family
	}
	if get("style:use-window-font-color") != "true" {
		run.Color = parseColor(get("fo:color"))
	}
	return run
}

// sameStyle reports whether two runs are drawn alike.
func sameStyle(a, b outRun) bool {
	eqFloat := func(x, y *float64) bool { return (x == nil) == (y == nil) && (x == nil || *x == *y) }
	eqString := func(x, y *string) bool { return (x == nil) == (y == nil) && (x == nil || *x == *y) }
	return a.Bold == b.Bold && a.Italic == b.Italic && a.Underline == b.Underline &&
		a.Strikethrough == b.Strikethrough && a.Color == b.Color &&
		eqFloat(a.FontSize, b.FontSize) && eqString(a.FontFamily, b.FontFamily)
}

// visibleText reports whether any paragraph shows something.
func visibleText(paragraphs []outParagraph) bool {
	for _, p := range paragraphs {
		for _, r := range p.Runs {
			if strings.TrimSpace(r.Text) != "" {
				return true
			}
		}
	}
	return false
}
