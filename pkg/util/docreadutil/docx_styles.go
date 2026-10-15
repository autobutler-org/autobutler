package docreadutil

// cspell:ignore abstractNum abstractNumId dstrike ilvl jc lvl lvlOverride lvlText numFmt numId numPr outlineLvl pPr pStyle rFonts rPr rPrDefault rStyle startOverride styleId vertAlign Consolas Menlo andale cascadia dejavu fira lucida roboto Kashida

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/officezip"
)

// maxStyleChain bounds how many styles a basedOn chain follows.
const maxStyleChain = 32

// maxListStart bounds a list's start number, so counting from it stays
// cheap whatever a document declares.
const maxListStart = 1_000_000

// monospaceFonts are the font families a run is shown in as code.
var monospaceFonts = map[string]bool{
	"andale mono": true, "cascadia code": true, "cascadia mono": true, "consolas": true,
	"courier": true, "courier new": true, "dejavu sans mono": true, "fira code": true,
	"fira mono": true, "liberation mono": true, "lucida console": true, "lucida sans typewriter": true,
	"menlo": true, "monaco": true, "roboto mono": true, "source code pro": true, "ubuntu mono": true,
}

// emptyStyles is the styles of a document with no styles part.
func emptyStyles() *docxStyles {
	return &docxStyles{byID: map[string]*docxStyle{}, resolved: map[string]*resolvedStyle{}}
}

// emptyNumbering is the numbering of a document with no numbering part.
func emptyNumbering() *docxNumbering {
	return &docxNumbering{abstract: map[string]*numLevels{}, lists: map[string]*numDef{}, counters: map[string]*listCounter{}}
}

// readStyles reads and holds styles.xml.
func readStyles(pkg *officezip.Package, part string, limits Limits) (*docxStyles, error) {
	dec, closer, err := heldXML(pkg, part, limits.MaxHeldBytes)
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	if _, err := root(dec, part, "styles"); err != nil {
		return nil, err
	}
	s := emptyStyles()
	count := 0
	err = children(dec, func(c xml.StartElement) error {
		switch c.Name.Local {
		case "docDefaults":
			return children(dec, func(c xml.StartElement) error {
				if c.Name.Local != "rPrDefault" {
					return dec.Skip()
				}
				return children(dec, func(c xml.StartElement) error {
					if c.Name.Local != "rPr" {
						return dec.Skip()
					}
					props, _, err := readRunProps(dec)
					s.defaults = props
					return err
				})
			})
		case "style":
			if count++; count > limits.MaxDefinitions {
				return fmt.Errorf("%w: %s declares more than %d styles", ErrTooLarge, part, limits.MaxDefinitions)
			}
			id := attr(c, "styleId")
			st, err := readStyle(dec, c)
			if err != nil {
				return err
			}
			if _, dup := s.byID[id]; !dup {
				s.byID[id] = st
			}
			// Unlike a property, an absent w:default is off.
			if def := attr(c, "default"); st.kind == "paragraph" && def != "" && onOff(def) == optOn && s.defaultPara == "" {
				s.defaultPara = id
			}
			return nil
		}
		return dec.Skip()
	})
	return s, err
}

// readStyle reads one w:style.
func readStyle(src tokenSource, start xml.StartElement) (*docxStyle, error) {
	st := &docxStyle{kind: attr(start, "type"), para: paraProps{outline: -1}}
	err := children(src, func(c xml.StartElement) error {
		var err error
		switch c.Name.Local {
		case "name":
			st.name = attr(c, "val")
		case "basedOn":
			st.basedOn = attr(c, "val")
		case "pPr":
			st.para, err = readParaProps(src)
			return err
		case "rPr":
			st.run, _, err = readRunProps(src)
			return err
		}
		return src.Skip()
	})
	return st, err
}

// resolve returns a style with its basedOn chain applied. An unknown id, a
// chain that loops and one longer than maxStyleChain all end where they
// stop making sense.
func (s *docxStyles) resolve(id string) *resolvedStyle {
	return s.resolveChain(id, 0)
}

func (s *docxStyles) resolveChain(id string, depth int) *resolvedStyle {
	if res, ok := s.resolved[id]; ok {
		return res
	}
	st := s.byID[id]
	if id == "" || st == nil || depth > maxStyleChain {
		return &resolvedStyle{para: paraProps{outline: -1}}
	}
	// A placeholder first, so a chain that loops back here ends.
	s.resolved[id] = &resolvedStyle{para: paraProps{outline: -1}}
	base := s.resolveChain(st.basedOn, depth+1)
	res := &resolvedStyle{run: base.run.over(st.run), para: base.para.over(st.para), heading: base.heading}
	switch level := headingForName(st.name); {
	case level > 0:
		res.heading = level
	case st.para.outline >= 0:
		res.heading = headingForOutline(st.para.outline)
	}
	s.resolved[id] = res
	return res
}

// headingForName is the heading level a built-in style's name gives: Title
// and "heading 1" to "heading 9", the last three shown as level 6.
func headingForName(name string) int {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "title" {
		return 1
	}
	rest, ok := strings.CutPrefix(name, "heading ")
	if !ok {
		return 0
	}
	n, err := strconv.Atoi(rest)
	if err != nil || n < 1 || n > 9 {
		return 0
	}
	return min(n, 6)
}

// headingForOutline is the heading level of an outline level: 0 to 8 are
// headings 1 to 6, the deepest shown as 6, and 9 is body text.
func headingForOutline(outline int) int {
	if outline < 0 || outline > 8 {
		return 0
	}
	return min(outline+1, 6)
}

// readParaProps reads a w:pPr.
func readParaProps(src tokenSource) (paraProps, error) {
	p := paraProps{outline: -1}
	err := children(src, func(c xml.StartElement) error {
		switch c.Name.Local {
		case "pStyle":
			p.style = attr(c, "val")
		case "numPr":
			return children(src, func(c xml.StartElement) error {
				switch c.Name.Local {
				case "ilvl":
					p.ilvl, p.hasIlvl = clampAtoi(attr(c, "val"), 0, 8), true
				case "numId":
					p.numID, p.hasNum = attr(c, "val"), true
				}
				return src.Skip()
			})
		case "outlineLvl":
			p.outline = clampAtoi(attr(c, "val"), 0, 9)
		case "jc":
			p.align = alignment(attr(c, "val"))
		case "pageBreakBefore":
			p.pageBreakBefore = onOff(attr(c, "val")) == optOn
		}
		// Everything else, w:pPrChange's record of what a tracked change
		// replaced among it, is skipped.
		return src.Skip()
	})
	return p, err
}

// over is p with what o sets laid over it.
func (p paraProps) over(o paraProps) paraProps {
	if o.style != "" {
		p.style = o.style
	}
	if o.outline >= 0 {
		p.outline = o.outline
	}
	if o.hasNum {
		p.numID, p.hasNum = o.numID, true
	}
	if o.hasIlvl {
		p.ilvl, p.hasIlvl = o.ilvl, true
	}
	if o.align != "" {
		p.align = o.align
	}
	p.pageBreakBefore = p.pageBreakBefore || o.pageBreakBefore
	return p
}

// readRunProps reads a w:rPr, and the character style it names.
func readRunProps(src tokenSource) (runProps, string, error) {
	var p runProps
	var style string
	err := children(src, func(c xml.StartElement) error {
		val := attr(c, "val")
		switch c.Name.Local {
		case "rStyle":
			style = val
		case "b":
			p.bold = onOff(val)
		case "i":
			p.italic = onOff(val)
		case "strike", "dstrike":
			p.strike = onOff(val)
		case "u":
			p.underline = optOn
			if val == "none" {
				p.underline = optOff
			}
		case "vertAlign":
			p.sup, p.sub = optOff, optOff
			switch val {
			case "superscript":
				p.sup = optOn
			case "subscript":
				p.sub = optOn
			}
		case "rFonts":
			font := attr(c, "ascii")
			if font == "" {
				font = attr(c, "hAnsi")
			}
			if font != "" {
				p.code = optOff
				if monospaceFonts[strings.ToLower(strings.TrimSpace(font))] {
					p.code = optOn
				}
			}
		case "vanish":
			p.hidden = onOff(val)
		}
		// Everything else, w:rPrChange among it, is skipped.
		return src.Skip()
	})
	return p, style, err
}

// over is p with what o sets laid over it.
func (p runProps) over(o runProps) runProps {
	pick := func(a, b opt) opt {
		if b != optUnset {
			return b
		}
		return a
	}
	return runProps{
		bold: pick(p.bold, o.bold), italic: pick(p.italic, o.italic), underline: pick(p.underline, o.underline),
		strike: pick(p.strike, o.strike), code: pick(p.code, o.code), sup: pick(p.sup, o.sup),
		sub: pick(p.sub, o.sub), hidden: pick(p.hidden, o.hidden),
	}
}

// run is a run of text with these properties.
func (p runProps) run(text, link string) Run {
	return Run{
		Text: text, Bold: p.bold == optOn, Italic: p.italic == optOn, Underline: p.underline == optOn,
		Strike: p.strike == optOn, Code: p.code == optOn, Superscript: p.sup == optOn,
		Subscript: p.sub == optOn && p.sup != optOn, Link: link,
	}
}

// onOff reads an OOXML on/off value; an absent value is on.
func onOff(val string) opt {
	switch val {
	case "0", "false", "off", "none":
		return optOff
	}
	return optOn
}

// alignment maps w:jc to a block's alignment.
func alignment(val string) string {
	switch val {
	case "center":
		return "center"
	case "right", "end":
		return "right"
	case "both", "distribute", "lowKashida", "mediumKashida", "highKashida", "thaiDistribute":
		return "justify"
	}
	return ""
}

// clampAtoi parses an integer and clamps it to [lo, hi]; what does not
// parse is lo.
func clampAtoi(s string, lo, hi int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return lo
	}
	return min(max(n, lo), hi)
}

// readNumbering reads and holds numbering.xml.
func readNumbering(pkg *officezip.Package, part string, limits Limits) (*docxNumbering, error) {
	dec, closer, err := heldXML(pkg, part, limits.MaxHeldBytes)
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	if _, err := root(dec, part, "numbering"); err != nil {
		return nil, err
	}
	n := emptyNumbering()
	count := 0
	err = children(dec, func(c xml.StartElement) error {
		switch c.Name.Local {
		case "abstractNum", "num":
			if count++; count > limits.MaxDefinitions {
				return fmt.Errorf("%w: %s declares more than %d lists", ErrTooLarge, part, limits.MaxDefinitions)
			}
		}
		switch c.Name.Local {
		case "abstractNum":
			levels := &numLevels{}
			if _, dup := n.abstract[attr(c, "abstractNumId")]; !dup {
				n.abstract[attr(c, "abstractNumId")] = levels
			}
			return children(dec, func(c xml.StartElement) error {
				if c.Name.Local != "lvl" {
					return dec.Skip()
				}
				ilvl := clampAtoi(attr(c, "ilvl"), 0, 8)
				lvl, err := readLevel(dec)
				levels[ilvl] = lvl
				return err
			})
		case "num":
			def := &numDef{starts: [9]int{-1, -1, -1, -1, -1, -1, -1, -1, -1}}
			if _, dup := n.lists[attr(c, "numId")]; !dup {
				n.lists[attr(c, "numId")] = def
			}
			return children(dec, func(c xml.StartElement) error {
				switch c.Name.Local {
				case "abstractNumId":
					def.abstract = attr(c, "val")
				case "lvlOverride":
					ilvl := clampAtoi(attr(c, "ilvl"), 0, 8)
					return children(dec, func(c xml.StartElement) error {
						switch c.Name.Local {
						case "startOverride":
							def.starts[ilvl] = clampAtoi(attr(c, "val"), 0, maxListStart)
						case "lvl":
							lvl, err := readLevel(dec)
							def.overrides[ilvl] = lvl
							return err
						}
						return dec.Skip()
					})
				}
				return dec.Skip()
			})
		}
		return dec.Skip()
	})
	return n, err
}

// readLevel reads one w:lvl. A level without w:start starts at 0, as the
// format says.
func readLevel(src tokenSource) (*numLevel, error) {
	lvl := &numLevel{format: "decimal"}
	err := children(src, func(c xml.StartElement) error {
		switch c.Name.Local {
		case "start":
			lvl.start = clampAtoi(attr(c, "val"), 0, maxListStart)
		case "numFmt":
			lvl.format = attr(c, "val")
		case "lvlText":
			lvl.text = attr(c, "val")
		}
		return src.Skip()
	})
	return lvl, err
}

// next numbers one more item of a list at a level, and returns its marker
// and whether the list is numbered. It reports false for a list the
// document does not define.
func (n *docxNumbering) next(numID string, ilvl int) (marker string, ordered, ok bool) {
	def := n.lists[numID]
	if def == nil {
		return "", false, false
	}
	abstract := n.abstract[def.abstract]
	level := func(i int) *numLevel {
		if def.overrides[i] != nil {
			return def.overrides[i]
		}
		if abstract != nil {
			return abstract[i]
		}
		return nil
	}
	start := func(i int) int {
		if def.starts[i] >= 0 {
			return def.starts[i]
		}
		if l := level(i); l != nil {
			return l.start
		}
		return 0
	}
	c := n.counters[numID]
	if c == nil {
		c = &listCounter{}
		n.counters[numID] = c
	}
	if c.started[ilvl] {
		c.value[ilvl]++
	} else {
		c.value[ilvl], c.started[ilvl] = start(ilvl), true
	}
	for i := ilvl + 1; i < len(c.started); i++ {
		c.started[i] = false
	}

	lvl := level(ilvl)
	switch {
	case lvl == nil || lvl.format == "bullet":
		text := ""
		if lvl != nil {
			text = lvl.text
		}
		return bulletMarker(text), false, true
	case lvl.format == "none":
		return "", false, true
	}
	var sb strings.Builder
	runes := []rune(lvl.text)
	for i := 0; i < len(runes); i++ {
		if runes[i] == '%' && i+1 < len(runes) && runes[i+1] >= '1' && runes[i+1] <= '9' {
			at := int(runes[i+1] - '1')
			value := start(at)
			if c.started[at] {
				value = c.value[at]
			}
			format := "decimal"
			if l := level(at); l != nil {
				format = l.format
			}
			sb.WriteString(formatNumber(value, format))
			i++
			continue
		}
		sb.WriteRune(runes[i])
	}
	return sb.String(), true, true
}

// bulletMarker is a bullet level's marker. Word writes most bullets as a
// character of the Symbol or Wingdings font, from the private use area, which
// no other font draws; those become a plain bullet.
func bulletMarker(text string) string {
	switch text {
	case "", "", "", "", "", "":
		return "•"
	case "o":
		return "◦"
	}
	for _, r := range text {
		if r >= 0xE000 && r <= 0xF8FF {
			return "•"
		}
	}
	return text
}

// formatNumber writes a list counter in a w:numFmt.
func formatNumber(n int, format string) string {
	switch format {
	case "decimalZero":
		return fmt.Sprintf("%02d", n)
	case "lowerLetter", "upperLetter":
		// Word counts a, b, … z, aa, bb, …; past zzz… it is a number.
		if n < 1 || n > 26*30 {
			break
		}
		letters := strings.Repeat(string(rune('a'+(n-1)%26)), (n-1)/26+1)
		if format == "upperLetter" {
			return strings.ToUpper(letters)
		}
		return letters
	case "lowerRoman", "upperRoman":
		if n < 1 || n > 3999 {
			break
		}
		if format == "upperRoman" {
			return strings.ToUpper(roman(n))
		}
		return roman(n)
	case "ordinal":
		return strconv.Itoa(n) + ordinalSuffix(n)
	}
	return strconv.Itoa(n)
}

// roman writes 1 to 3999 in lowercase Roman numerals.
func roman(n int) string {
	values := []int{1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1}
	numerals := []string{"m", "cm", "d", "cd", "c", "xc", "l", "xl", "x", "ix", "v", "iv", "i"}
	var sb strings.Builder
	for i, v := range values {
		for n >= v {
			sb.WriteString(numerals[i])
			n -= v
		}
	}
	return sb.String()
}

// ordinalSuffix is the English ordinal suffix of n: st, nd, rd or th.
func ordinalSuffix(n int) string {
	if n%100 >= 11 && n%100 <= 13 {
		return "th"
	}
	switch n % 10 {
	case 1:
		return "st"
	case 2:
		return "nd"
	case 3:
		return "rd"
	}
	return "th"
}

// readTitle reads the title from a package's properties part, whose root
// has the local name rootName and holds the title directly or, when
// container is not empty, inside a child of that name. A part that cannot
// be read has none: the title is a nicety, never a reason to fail.
func readTitle(pkg *officezip.Package, part string, maxBytes int64, rootName, container string) string {
	dec, closer, err := heldXML(pkg, part, maxBytes)
	if err != nil {
		return ""
	}
	defer closer.Close()
	if _, err := root(dec, part, rootName); err != nil {
		return ""
	}
	title := ""
	var find func(c xml.StartElement) error
	find = func(c xml.StartElement) error {
		switch {
		case container != "" && c.Name.Local == container:
			return children(dec, find)
		case c.Name.Local != "title" || title != "":
			return dec.Skip()
		}
		t, err := text(dec)
		title = strings.TrimSpace(t)
		return err
	}
	_ = children(dec, find)
	if r := []rune(title); len(r) > maxTitleRunes {
		title = string(r[:maxTitleRunes])
	}
	return title
}

// maxTitleRunes bounds the title the meta line carries.
const maxTitleRunes = 500
