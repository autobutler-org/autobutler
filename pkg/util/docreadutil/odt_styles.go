package docreadutil

// cspell:ignore oblique

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// The style families the ODT reader reads; the rest describe tables,
// frames and pages, which it does not draw.
const (
	familyParagraph = "paragraph"
	familyText      = "text"
)

// newOdtStyles returns an empty style set bounded by limits.
func newOdtStyles(limits Limits) *odtStyles {
	return &odtStyles{
		limits:    limits,
		common:    map[string]*odtStyle{},
		automatic: map[string]*odtStyle{},
		defaults:  map[string]*odtStyle{},
		lists:     map[string]*odtListStyle{},
		fixed:     map[string]bool{},
		resolved:  map[string]*odtResolved{},
	}
}

// define counts one more definition, and the bytes of its names, against
// the limits a held part has.
func (s *odtStyles) define(names ...string) error {
	if s.count++; s.count > s.limits.MaxDefinitions {
		return fmt.Errorf("%w: the document declares more than %d styles", ErrTooLarge, s.limits.MaxDefinitions)
	}
	for _, n := range names {
		s.bytes += int64(len(n))
	}
	if s.bytes > s.limits.MaxHeldBytes {
		return fmt.Errorf("%w: the document's style names are larger than %d bytes", ErrTooLarge, s.limits.MaxHeldBytes)
	}
	return nil
}

// readFontFaces reads an office:font-face-decls, noting which faces are of
// fixed pitch.
func (s *odtStyles) readFontFaces(src tokenSource) error {
	return children(src, func(c xml.StartElement) error {
		if c.Name.Local == "font-face" {
			name := attr(c, "name")
			if err := s.define(name); err != nil {
				return err
			}
			if attr(c, "font-pitch") == "fixed" || monospaceFonts[unquote(attr(c, "font-family"))] {
				s.fixed[name] = true
			}
		}
		return src.Skip()
	})
}

// readStyles reads an office:styles or office:automatic-styles.
func (s *odtStyles) readStyles(src tokenSource, automatic bool) error {
	return children(src, func(c xml.StartElement) error {
		switch c.Name.Local {
		case "style", "default-style":
			family := attr(c, "family")
			if family != familyParagraph && family != familyText {
				return src.Skip()
			}
			name := attr(c, "name")
			if err := s.define(name, attr(c, "parent-style-name"), attr(c, "list-style-name")); err != nil {
				return err
			}
			st, err := s.readStyle(src, c)
			if err != nil {
				return err
			}
			key := family + ":" + name
			switch {
			case c.Name.Local == "default-style":
				s.defaults[family] = st
			case automatic:
				s.automatic[key] = st
			default:
				s.common[key] = st
			}
			return nil
		case "list-style":
			name := attr(c, "name")
			if err := s.define(name); err != nil {
				return err
			}
			ls, err := readOdtListStyle(src)
			if err != nil {
				return err
			}
			if _, dup := s.lists[name]; !dup {
				s.lists[name] = ls
			}
			return nil
		}
		return src.Skip()
	})
}

// readStyle reads one style:style or style:default-style.
func (s *odtStyles) readStyle(src tokenSource, start xml.StartElement) (*odtStyle, error) {
	st := &odtStyle{
		parent:    attr(start, "parent-style-name"),
		listStyle: attr(start, "list-style-name"),
		outline:   clampAtoi(attr(start, "default-outline-level"), 0, 10),
	}
	name := attr(start, "display-name")
	if name == "" {
		name = strings.ReplaceAll(attr(start, "name"), "_20_", " ")
	}
	st.heading = headingForName(name)
	err := children(src, func(c xml.StartElement) error {
		switch c.Name.Local {
		case "text-properties":
			st.run = odtTextProps(c)
			st.run.code = s.fontCode(c)
		case "paragraph-properties":
			if align, ok := odtAlignment(attr(c, "text-align")); ok {
				st.align, st.alignSet = align, true
			}
			st.breakBefore = odtBreak(attr(c, "break-before"))
			st.breakAft = odtBreak(attr(c, "break-after"))
		}
		return src.Skip()
	})
	return st, err
}

// odtTextProps reads a style:text-properties.
func odtTextProps(c xml.StartElement) runProps {
	var p runProps
	for _, a := range c.Attr {
		v := strings.TrimSpace(a.Value)
		switch a.Name.Local {
		case "font-weight":
			n, err := strconv.Atoi(v)
			p.bold = boolOpt(v == "bold" || (err == nil && n >= 600))
		case "font-style":
			p.italic = boolOpt(v == "italic" || v == "oblique")
		case "text-underline-style":
			p.underline = boolOpt(v != "none")
		case "text-line-through-style":
			p.strike = boolOpt(v != "none")
		case "text-position":
			p.sup, p.sub = textPosition(v)
		case "display":
			p.hidden = boolOpt(v == "none")
		}
	}
	return p
}

// fontCode reads whether a style:text-properties names a monospace font,
// by face or by family. A part declares its font faces before its styles.
func (s *odtStyles) fontCode(c xml.StartElement) opt {
	code := optUnset
	if name := attr(c, "font-name"); name != "" {
		code = boolOpt(s.fixed[name] || monospaceFonts[strings.ToLower(name)])
	}
	if family := attr(c, "font-family"); family != "" {
		code = boolOpt(monospaceFonts[unquote(family)])
	}
	return code
}

// boolOpt is on or off.
func boolOpt(on bool) opt {
	if on {
		return optOn
	}
	return optOff
}

// textPosition reads style:text-position: "super", "sub", or a percentage
// raising or lowering the text, then an optional size.
func textPosition(v string) (sup, sub opt) {
	first, _, _ := strings.Cut(v, " ")
	switch first {
	case "super":
		return optOn, optOff
	case "sub":
		return optOff, optOn
	}
	n, err := strconv.ParseFloat(strings.TrimSuffix(first, "%"), 64)
	switch {
	case err != nil:
		return optUnset, optUnset
	case n > 0:
		return optOn, optOff
	case n < 0:
		return optOff, optOn
	}
	return optOff, optOff
}

// odtAlignment maps fo:text-align to a block's alignment, and reports
// whether the value is one.
func odtAlignment(v string) (string, bool) {
	switch v {
	case "start", "left":
		return "", true
	case "center":
		return "center", true
	case "end", "right":
		return "right", true
	case "justify":
		return "justify", true
	}
	return "", false
}

// odtBreak reads fo:break-before or fo:break-after.
func odtBreak(v string) opt {
	switch v {
	case "page":
		return optOn
	case "auto", "column":
		return optOff
	}
	return optUnset
}

// unquote is a font family's name, lowercased, without its quotes.
func unquote(family string) string {
	first, _, _ := strings.Cut(family, ",")
	return strings.ToLower(strings.Trim(strings.TrimSpace(first), `'"`))
}

// readOdtListStyle reads one text:list-style's levels.
func readOdtListStyle(src tokenSource) (*odtListStyle, error) {
	var ls odtListStyle
	err := children(src, func(c xml.StartElement) error {
		level := clampAtoi(attr(c, "level"), 1, odtLevels) - 1
		l := &odtListLevel{
			prefix: attr(c, "num-prefix"), suffix: attr(c, "num-suffix"),
			start:   clampAtoi(attr(c, "start-value"), 0, maxListStart),
			display: clampAtoi(attr(c, "display-levels"), 1, level+1),
		}
		if attr(c, "start-value") == "" {
			l.start = 1
		}
		switch c.Name.Local {
		case "list-level-style-number":
			l.numbered, l.format = true, attr(c, "num-format")
			l.letterSync = attr(c, "num-letter-sync") == "true"
		case "list-level-style-bullet":
			l.bullet = attr(c, "bullet-char")
		case "list-level-style-image":
		default:
			return src.Skip()
		}
		if ls[level] == nil {
			ls[level] = l
		}
		return src.Skip()
	})
	return &ls, err
}

// paragraph resolves a paragraph style by name; the empty name is the
// default style alone.
func (s *odtStyles) paragraph(name string) *odtResolved {
	return s.resolve(familyParagraph, name)
}

// text resolves a text style's run formatting, without the default style,
// which the paragraph it sits in already carries.
func (s *odtStyles) text(name string) runProps {
	if name == "" {
		return runProps{}
	}
	return s.resolveChain(familyText, name, true, 0).run
}

// resolve returns a style with its parents and its family's default
// applied.
func (s *odtStyles) resolve(family, name string) *odtResolved {
	key := "r:" + family + ":" + name
	if res, ok := s.resolved[key]; ok {
		return res
	}
	res := *s.resolveChain(family, name, true, 0)
	if def := s.defaults[family]; def != nil {
		res.run = def.run.over(res.run)
	}
	s.resolved[key] = &res
	return &res
}

// resolveChain applies a style over its parents. A name looks among the
// automatic styles first when automatic is set; a parent is always a common
// style. A chain that loops, or runs past maxStyleChain, ends there.
func (s *odtStyles) resolveChain(family, name string, automatic bool, depth int) *odtResolved {
	key := family + ":" + name
	cacheKey := "c:" + key
	st := s.common[key]
	if automatic && s.automatic[key] != nil {
		st, cacheKey = s.automatic[key], "a:"+key
	}
	if res, ok := s.resolved[cacheKey]; ok {
		return res
	}
	if name == "" || st == nil || depth > maxStyleChain {
		return &odtResolved{}
	}
	// A placeholder first, so a chain that loops back here ends.
	s.resolved[cacheKey] = &odtResolved{}
	base := *s.resolveChain(family, st.parent, false, depth+1)
	res := &base
	res.run = base.run.over(st.run)
	if st.heading > 0 {
		res.heading = st.heading
	}
	if st.outline > 0 {
		res.outline = st.outline
	}
	if st.alignSet {
		res.align = st.align
	}
	if st.listStyle != "" {
		res.listStyle = st.listStyle
	}
	if st.breakBefore != optUnset {
		res.breakBefore = st.breakBefore == optOn
	}
	if st.breakAft != optUnset {
		res.breakAfter = st.breakAft == optOn
	}
	s.resolved[cacheKey] = res
	return res
}

// marker numbers one more item at level of the list counted by c, in list
// style ls, and returns its marker and whether the list is numbered. start
// is the item's own text:start-value, if it sets one.
func (c *odtCounter) marker(ls *odtListStyle, level int, start string) (string, bool) {
	lvl := func(i int) *odtListLevel {
		if ls == nil {
			return nil
		}
		return ls[i]
	}
	first := func(i int) int {
		if l := lvl(i); l != nil {
			return l.start
		}
		return 1
	}
	switch {
	case start != "":
		c.value[level] = clampAtoi(start, 0, maxListStart)
	case c.started[level]:
		c.value[level]++
	default:
		c.value[level] = first(level)
	}
	c.started[level] = true
	for i := level + 1; i < odtLevels; i++ {
		c.started[i] = false
	}

	l := lvl(level)
	switch {
	case l == nil || !l.numbered:
		bullet := ""
		if l != nil {
			bullet = l.bullet
		}
		return bulletMarker(bullet), false
	case l.format == "":
		return strings.TrimSpace(l.prefix + l.suffix), false
	}
	numbers := make([]string, 0, l.display)
	for i := level - l.display + 1; i <= level; i++ {
		value := first(i)
		if c.started[i] {
			value = c.value[i]
		}
		format, sync := "1", false
		if li := lvl(i); li != nil && li.numbered && li.format != "" {
			format, sync = li.format, li.letterSync
		}
		numbers = append(numbers, odtNumber(value, format, sync))
	}
	return l.prefix + strings.Join(numbers, ".") + l.suffix, true
}

// odtNumber writes a list counter in a style:num-format. Letters run a, b,
// … z, aa, ab, … unless sync asks for a, … z, aa, bb, as Word counts.
func odtNumber(n int, format string, sync bool) string {
	switch format {
	case "a", "A":
		if sync || n < 1 {
			if format == "a" {
				return formatNumber(n, "lowerLetter")
			}
			return formatNumber(n, "upperLetter")
		}
		var letters []byte
		for ; n > 0; n = (n - 1) / 26 {
			letters = append([]byte{byte('a' + (n-1)%26)}, letters...)
		}
		if format == "A" {
			return strings.ToUpper(string(letters))
		}
		return string(letters)
	case "i":
		return formatNumber(n, "lowerRoman")
	case "I":
		return formatNumber(n, "upperRoman")
	}
	return strconv.Itoa(n)
}
