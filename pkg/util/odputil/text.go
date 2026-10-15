package odputil

import (
	"encoding/xml"
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/autobutler-org/quark/pkg/util/officezip"
)

// The editor's defaults for what a run or paragraph leaves unset.
const (
	// defaultFontPt is the size of text no style sizes, Impress's default.
	defaultFontPt = 18.0
	// defaultLineSpacing is a paragraph's line height as a multiple of its
	// font size at single spacing, which is how the editor and pptxutil
	// both measure it: ODF's 100% is the same single spacing.
	defaultLineSpacing = 1.2
	// maxSpaces bounds the spaces one text:s stands for.
	maxSpaces = 1_000
)

// cmPerPt converts points to cm.
const cmPerPt = 2.54 / 72

// textContext is what a text body's paragraphs inherit from the element
// that holds them.
type textContext struct {
	refs []styleRef
	// outline is set in an outline placeholder, whose paragraphs are
	// bulleted by its presentation style without being in a text:list.
	outline bool
}

// listState is where a paragraph sits in a text:list: its list style and
// how deep it is.
type listState struct {
	style string
	level int
	// header is set inside a text:list-header, whose paragraphs have no
	// marker.
	header bool
}

// runStyle is a run's resolved style; two runs of the same style are one.
type runStyle struct {
	bold, italic, underline, strike, hidden bool
	size                                    float64
	family, color                           string
}

// paraBuilder collects a paragraph's runs, and the paragraphs a line break
// starts.
type paraBuilder struct {
	s        *slideReader
	template outParagraph
	out      []outParagraph
	// space is set after collapsible white space, and at a paragraph's
	// start, where white space is dropped.
	space bool
	// pending is the text of the run being built, in pendingStyle; it
	// joins the current paragraph's runs when the style changes or the
	// paragraph ends.
	pending      strings.Builder
	pendingStyle runStyle
}

// textContext is the context an element's text is read in: the element's
// own graphic and presentation styles, and the paragraph style its shape
// text takes.
func (s *slideReader) textContext(c xml.StartElement) textContext {
	return textContext{
		refs:    append(refs(c), styleRef{"paragraph", attr(c, nsDraw, "text-style-name")}),
		outline: attr(c, nsPresentation, "class") == "outline",
	}
}

// textBody reads the paragraphs of a draw:text-box or a table cell through
// to its end.
func (s *slideReader) textBody(dec *officezip.Decoder, ctx textContext) ([]outParagraph, error) {
	paragraphs := []outParagraph{}
	err := children(dec, func(c xml.StartElement) error {
		ps, err := s.textBlock(dec, c, ctx, nil)
		paragraphs = append(paragraphs, ps...)
		return err
	})
	return paragraphs, err
}

// textBlock reads one block of text: a paragraph, a heading or a list. Any
// other element is skipped.
func (s *slideReader) textBlock(dec *officezip.Decoder, c xml.StartElement, ctx textContext, list *listState) ([]outParagraph, error) {
	switch {
	case is(c, nsText, "p"), is(c, nsText, "h"):
		return s.paragraph(dec, c, ctx, list)
	case is(c, nsText, "list"):
		inner := listState{level: 1}
		if list != nil {
			inner = listState{style: list.style, level: list.level + 1}
		}
		if name := attr(c, nsText, "style-name"); name != "" {
			inner.style = name
		}
		var out []outParagraph
		err := children(dec, func(item xml.StartElement) error {
			if !is(item, nsText, "list-item") && !is(item, nsText, "list-header") {
				return skip(dec)
			}
			itemState := inner
			itemState.header = item.Name.Local == "list-header"
			return children(dec, func(block xml.StartElement) error {
				ps, err := s.textBlock(dec, block, ctx, &itemState)
				out = append(out, ps...)
				return err
			})
		})
		return out, err
	}
	return nil, skip(dec)
}

// paragraph reads one text:p or text:h, and the paragraphs its line breaks
// start.
func (s *slideReader) paragraph(dec *officezip.Decoder, c xml.StartElement, ctx textContext, list *listState) ([]outParagraph, error) {
	paraRefs := append([]styleRef{{"paragraph", attr(c, nsText, "style-name")}}, ctx.refs...)
	template := outParagraph{Runs: []outRun{}}
	if v, ok := s.styles.prop(s.scope, paraRefs, "fo:text-align"); ok {
		template.Align = map[string]string{"center": "center", "end": "end", "right": "end", "justify": "justify"}[v]
	}
	switch {
	case list != nil && !list.header:
		template.List = s.listKind(list, paraRefs)
	case list == nil && ctx.outline:
		template.List = s.listKind(&listState{level: 1}, paraRefs)
	}
	b := &paraBuilder{s: s, template: template, out: []outParagraph{cloneParagraph(template)}, space: true}
	if err := s.inline(dec, paraRefs, b); err != nil {
		return nil, err
	}
	b.flush()
	lineHeight, _ := s.styles.prop(s.scope, paraRefs, "fo:line-height")
	base := s.runStyle(paraRefs).size
	for i := range b.out {
		size := base
		if runs := b.out[i].Runs; len(runs) > 0 && runs[0].FontSize != nil {
			size = *runs[0].FontSize
		}
		b.out[i].LineSpacing = s.lineSpacing(lineHeight, size)
	}
	return b.out, nil
}

// listKind is the marker a list paragraph shows: its list style's at its
// level, or the one its paragraph or presentation style holds. A list no
// style describes is bulleted.
func (s *slideReader) listKind(list *listState, paraRefs []styleRef) string {
	level := min(max(list.level, 1), listLevels) - 1
	if ls := s.styles.listStyle(s.scope, list.style); ls != nil {
		return ls.levels[level]
	}
	kind, found := listBullet, false
	s.styles.each(s.scope, paraRefs, func(st *odfStyle) bool {
		switch {
		case st.listStyle != "":
			if ls := s.styles.listStyle(s.scope, st.listStyle); ls != nil {
				kind, found = ls.levels[level], true
			}
		case st.list != nil:
			kind, found = st.list.levels[level], true
		}
		return found
	})
	return kind
}

// lineSpacing is a paragraph's line height as a multiple of its font
// size, nil at single spacing, which is the editor's default.
func (s *slideReader) lineSpacing(lineHeight string, size float64) *float64 {
	var spacing float64
	if pct, ok := parsePercent(lineHeight); ok {
		spacing = pct * defaultLineSpacing
	} else if cm, ok := parseCm(lineHeight); ok && size > 0 {
		spacing = s.units(cm) / size
	} else {
		return nil
	}
	spacing = round(spacing)
	if spacing <= 0 || math.Abs(spacing-defaultLineSpacing) < 0.005 {
		return nil
	}
	return &spacing
}

// inline reads the inline content of a paragraph or span through to its
// end: text, spans, spaces, tabs and line breaks. A field shows the text
// the file holds for it; a note, an annotation or a change mark is left
// out.
func (s *slideReader) inline(dec *officezip.Decoder, r []styleRef, b *paraBuilder) error {
	style := s.runStyle(r)
	for {
		tok, err := dec.Token()
		if err != nil {
			return xmlError(err)
		}
		switch t := tok.(type) {
		case xml.CharData:
			if err := b.collapsed(string(t), style); err != nil {
				return err
			}
		case xml.StartElement:
			if err := s.inlineElement(dec, t, r, style, b); err != nil {
				return err
			}
		case xml.EndElement:
			return nil
		}
	}
}

// inlineElement reads one element inside a paragraph.
func (s *slideReader) inlineElement(dec *officezip.Decoder, t xml.StartElement, r []styleRef, style runStyle, b *paraBuilder) error {
	if t.Name.Space == nsText {
		switch t.Name.Local {
		case "span", "a":
			if name := attr(t, nsText, "style-name"); name != "" {
				r = append([]styleRef{{"text", name}}, r...)
			}
			return s.inline(dec, r, b)
		case "s":
			n := 1
			if v, err := strconv.Atoi(attr(t, nsText, "c")); err == nil {
				n = min(max(v, 1), maxSpaces)
			}
			if err := b.raw(strings.Repeat(" ", n), style); err != nil {
				return err
			}
			return skip(dec)
		case "tab":
			if err := b.raw("\t", style); err != nil {
				return err
			}
			return skip(dec)
		case "line-break":
			b.flush()
			b.out = append(b.out, cloneParagraph(b.template))
			b.space = true
			return skip(dec)
		case "note", "soft-page-break", "bookmark", "bookmark-start", "bookmark-end", "change",
			"change-start", "change-end", "reference-mark", "reference-mark-start", "reference-mark-end",
			"toc-mark", "alphabetical-index-mark", "ruby-text":
			return skip(dec)
		}
	}
	if t.Name.Space == nsOffice && (t.Name.Local == "annotation" || t.Name.Local == "annotation-end") {
		return skip(dec)
	}
	return s.inline(dec, r, b)
}

// runStyle resolves the style of text along r's chains.
func (s *slideReader) runStyle(r []styleRef) runStyle {
	get := func(key string) string {
		v, _ := s.styles.prop(s.scope, r, key)
		return strings.TrimSpace(v)
	}
	style := runStyle{color: parseColor(get("fo:color")), hidden: get("text:display") == "none"}
	weight := get("fo:font-weight")
	n, err := strconv.Atoi(weight)
	style.bold = weight == "bold" || (err == nil && n >= 600)
	italic := get("fo:font-style")
	style.italic = italic == "italic" || italic == "oblique"
	underline, strike := get("style:text-underline-style"), get("style:text-line-through-style")
	style.underline = underline != "" && underline != "none"
	style.strike = strike != "" && strike != "none"

	s.styles.each(s.scope, r, func(st *odfStyle) bool {
		if name, ok := st.props["style:font-name"]; ok {
			style.family = s.styles.fonts[name]
			if style.family == "" {
				style.family = unquoteFamily(name)
			}
			return true
		}
		if family, ok := st.props["fo:font-family"]; ok {
			style.family = unquoteFamily(family)
			return true
		}
		return false
	})

	// A percentage scales the size its parents set.
	factor, pt := 1.0, defaultFontPt
	s.styles.each(s.scope, r, func(st *odfStyle) bool {
		v, ok := st.props["fo:font-size"]
		if !ok {
			return false
		}
		if pct, ok := parsePercent(v); ok {
			factor *= pct
			return false
		}
		if cm, ok := parseCm(v); ok && cm > 0 {
			pt = cm / cmPerPt
			return true
		}
		return false
	})
	style.size = math.Max(s.units(pt*factor*cmPerPt), 0.5)
	return style
}

// run is a run of text in style.
func (st runStyle) run(text string) outRun {
	size := st.size
	run := outRun{
		Text: text, Bold: st.bold, Italic: st.italic, Underline: st.underline, Strikethrough: st.strike,
		FontSize: &size, Color: st.color,
	}
	if st.family != "" {
		family := st.family
		run.FontFamily = &family
	}
	return run
}

// collapsed adds character data, its white space collapsed as ODF
// collapses it: a run of spaces, tabs and newlines is one space, and none
// at a paragraph's start.
func (b *paraBuilder) collapsed(text string, style runStyle) error {
	var sb strings.Builder
	for _, r := range text {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !b.space {
				sb.WriteByte(' ')
				b.space = true
			}
			continue
		}
		if r == unicode.ReplacementChar || (r < ' ' && r != '\t') {
			continue
		}
		sb.WriteRune(r)
		b.space = false
	}
	return b.add(sb.String(), style)
}

// raw adds text that keeps its white space: a text:s or a text:tab.
func (b *paraBuilder) raw(text string, style runStyle) error {
	b.space = false
	return b.add(text, style)
}

// add appends text to the run being built, starting a new run when the
// style changes.
func (b *paraBuilder) add(text string, style runStyle) error {
	if text == "" || style.hidden {
		return nil
	}
	if err := b.s.countText(len(text)); err != nil {
		return err
	}
	if b.pending.Len() > 0 && b.pendingStyle != style {
		b.flush()
	}
	b.pendingStyle = style
	b.pending.WriteString(text)
	return nil
}

// flush ends the run being built, joining it to the current paragraph's
// last run when that is of the same style.
func (b *paraBuilder) flush() {
	if b.pending.Len() == 0 {
		return
	}
	text := b.pending.String()
	b.pending.Reset()
	cur := &b.out[len(b.out)-1]
	if n := len(cur.Runs); n > 0 && sameStyle(cur.Runs[n-1], b.pendingStyle) {
		cur.Runs[n-1].Text += text
		return
	}
	cur.Runs = append(cur.Runs, b.pendingStyle.run(text))
}

// sameStyle reports whether run is in style.
func sameStyle(run outRun, style runStyle) bool {
	family := ""
	if run.FontFamily != nil {
		family = *run.FontFamily
	}
	return run.Bold == style.bold && run.Italic == style.italic && run.Underline == style.underline &&
		run.Strikethrough == style.strike && run.FontSize != nil && *run.FontSize == style.size &&
		family == style.family && run.Color == style.color
}

// cloneParagraph is a copy of p with runs of its own.
func cloneParagraph(p outParagraph) outParagraph {
	p.Runs = []outRun{}
	return p
}

// hasText reports whether any paragraph shows something besides white
// space.
func hasText(paragraphs []outParagraph) bool {
	for _, p := range paragraphs {
		for _, r := range p.Runs {
			if strings.TrimSpace(r.Text) != "" {
				return true
			}
		}
	}
	return false
}

// textBox is the text box over frame that draws paragraphs, inset by the
// element's padding. anchor and fit are what the box is when its styles set
// neither: a shape's text sits in its middle and does not grow it.
func (s *slideReader) textBox(c xml.StartElement, frame outFrame, paragraphs []outParagraph, anchor, fit string) outText {
	r := refs(c)
	if paragraphs == nil {
		paragraphs = []outParagraph{}
	}
	box := outText{ID: s.id(), Type: typeText, Frame: s.inset(frame, r), Paragraphs: paragraphs, Anchor: anchor}
	if v, ok := s.styles.prop(s.scope, r, "draw:textarea-vertical-align"); ok {
		box.Anchor = map[string]string{"middle": "middle", "bottom": "bottom"}[v]
	}
	fitToSize, _ := s.styles.prop(s.scope, r, "draw:fit-to-size")
	shrink, _ := s.styles.prop(s.scope, r, "style:shrink-to-fit")
	grow, growSet := s.styles.prop(s.scope, r, "draw:auto-grow-height")
	box.AutoFit = fit
	switch {
	case fitToSize == "shrink-to-fit" || fitToSize == "true" || isTrue(shrink):
		box.AutoFit = "shrink"
	case growSet && grow == "false":
		box.AutoFit = "fixed"
	case growSet:
		box.AutoFit = ""
	}
	return box
}
