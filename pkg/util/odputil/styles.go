package odputil

// cspell:ignore smil xlink

import (
	"encoding/xml"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/officezip"
)

// Where a style is looked up: master pages see styles.xml's automatic
// styles, slides content.xml's, and both see styles.xml's common ones.
const (
	scopeMaster = iota
	scopeContent
	scopeCount
)

// maxStyleChain bounds how many parents a style lookup follows, which also
// ends a cycle of parents.
const maxStyleChain = 32

// listLevels is how many levels an ODF list style has.
const listLevels = 10

// The list kinds a .qslide paragraph has.
const (
	listBullet   = "bullet"
	listNumbered = "numbered"
)

// heldProperties are the style properties the reader draws, by prefix and
// local name. Nothing else a style sets is held.
var heldProperties = map[string]bool{
	// graphic-properties and drawing-page-properties
	"draw:fill": true, "draw:fill-color": true, "draw:fill-image-name": true, "draw:fill-gradient-name": true,
	"draw:fill-hatch-name": true, "draw:opacity": true, "draw:stroke": true, "svg:stroke-color": true,
	"svg:stroke-width": true, "draw:stroke-dash": true, "svg:stroke-opacity": true, "draw:marker-start": true,
	"draw:marker-end": true, "draw:textarea-vertical-align": true, "draw:auto-grow-height": true,
	"draw:fit-to-size": true, "style:shrink-to-fit": true, "fo:padding": true, "fo:padding-left": true,
	"fo:padding-right": true, "fo:padding-top": true, "fo:padding-bottom": true,
	"presentation:background-visible": true, "presentation:background-objects-visible": true,
	"presentation:transition-style": true, "presentation:transition-speed": true, "smil:type": true,
	"smil:subtype": true, "smil:direction": true,
	// paragraph-properties
	"fo:text-align": true, "fo:line-height": true,
	// text-properties
	"fo:font-size": true, "fo:font-weight": true, "fo:font-style": true, "style:text-underline-style": true,
	"style:text-line-through-style": true, "fo:color": true, "style:font-name": true, "fo:font-family": true,
	"text:display": true,
	// table-column-, table-row- and table-cell-properties
	"style:column-width": true, "style:row-height": true, "style:min-row-height": true,
	"fo:background-color": true, "fo:border": true, "fo:border-top": true, "fo:border-right": true,
	"fo:border-bottom": true, "fo:border-left": true, "style:vertical-align": true,
}

// odfStyle is one style:style or style:default-style, the properties the
// reader draws and where to look next.
type odfStyle struct {
	parent    string
	listStyle string
	// list is a list style held inside the style itself, as a presentation
	// outline style holds its bullets.
	list   *odfListStyle
	props  map[string]string
	common bool
}

// odfListStyle is a text:list-style: the kind of marker at each level, ""
// for none.
type odfListStyle struct {
	levels [listLevels]string
}

// styleRef names a style to look properties up in.
type styleRef struct {
	family, name string
}

// styleSet is every definition held from styles.xml and content.xml.
type styleSet struct {
	common   map[string]*odfStyle
	auto     [scopeCount]map[string]*odfStyle
	defaults map[string]*odfStyle
	// lists holds the list styles of styles.xml's office:styles, and
	// autoLists those of each part's automatic styles.
	lists     map[string]*odfListStyle
	autoLists [scopeCount]map[string]*odfListStyle
	// fonts maps a font face's name to its family.
	fonts map[string]string
	// fillImages, gradients, hatches and dashes are the named fills and
	// dashes office:styles declares: a fill image's picture, a gradient's
	// and a hatch's color, a dash's kind.
	fillImages map[string]string
	gradients  map[string]string
	hatches    map[string]string
	dashes     map[string]string
	// pageLayouts maps a page layout to its width and height in cm.
	pageLayouts map[string][2]float64
	count       int
	bytes       int64
}

// newStyleSet returns an empty style set.
func newStyleSet() *styleSet {
	st := &styleSet{
		common: map[string]*odfStyle{}, defaults: map[string]*odfStyle{}, lists: map[string]*odfListStyle{},
		fonts: map[string]string{}, fillImages: map[string]string{}, gradients: map[string]string{},
		hatches: map[string]string{}, dashes: map[string]string{}, pageLayouts: map[string][2]float64{},
	}
	for i := range st.auto {
		st.auto[i] = map[string]*odfStyle{}
		st.autoLists[i] = map[string]*odfListStyle{}
	}
	return st
}

// define counts one more definition, and the bytes it holds, against
// [MaxStyleDefinitions] and [MaxStyleBytes].
func (st *styleSet) define(held ...string) error {
	if st.count++; st.count > MaxStyleDefinitions {
		return fmt.Errorf("%w: the presentation declares more than %d styles", ErrTooLarge, MaxStyleDefinitions)
	}
	for _, s := range held {
		st.bytes += int64(len(s))
	}
	if st.bytes > MaxStyleBytes {
		return fmt.Errorf("%w: the presentation's styles hold more than %d bytes", ErrTooLarge, MaxStyleBytes)
	}
	return nil
}

// readFontFaces reads an office:font-face-decls.
func (st *styleSet) readFontFaces(dec *officezip.Decoder) error {
	return children(dec, func(c xml.StartElement) error {
		if is(c, nsStyle, "font-face") {
			name, family := attr(c, nsStyle, "name"), unquoteFamily(attr(c, nsSVG, "font-family"))
			if err := st.define(name, family); err != nil {
				return err
			}
			if name != "" && family != "" {
				st.fonts[name] = family
			}
		}
		return skip(dec)
	})
}

// readStyles reads an office:styles, or an office:automatic-styles of the
// part read in scope.
func (st *styleSet) readStyles(dec *officezip.Decoder, automatic bool, scope int) error {
	return children(dec, func(c xml.StartElement) error {
		name := attr(c, nsStyle, "name")
		if c.Name.Space == nsDraw {
			name = attr(c, nsDraw, "name")
		}
		switch {
		case is(c, nsStyle, "style"), is(c, nsStyle, "default-style"):
			s, err := st.readStyle(dec, c)
			if err != nil {
				return err
			}
			s.common = !automatic
			family := attr(c, nsStyle, "family")
			key := family + ":" + name
			switch {
			case c.Name.Local == "default-style":
				st.defaults[family] = s
			case automatic:
				st.auto[scope][key] = s
			default:
				st.common[key] = s
			}
			return nil
		case is(c, nsText, "list-style"):
			ls, err := st.readListStyle(dec, name)
			if err != nil {
				return err
			}
			if automatic {
				st.autoLists[scope][name] = ls
			} else {
				st.lists[name] = ls
			}
			return nil
		case is(c, nsStyle, "page-layout"):
			return st.readPageLayout(dec, name)
		case is(c, nsDraw, "fill-image"):
			return st.hold(dec, st.fillImages, name, attr(c, nsXlink, "href"))
		case is(c, nsDraw, "gradient"):
			return st.hold(dec, st.gradients, name, parseColor(attr(c, nsDraw, "start-color")))
		case is(c, nsDraw, "hatch"):
			return st.hold(dec, st.hatches, name, parseColor(attr(c, nsDraw, "color")))
		case is(c, nsDraw, "stroke-dash"):
			return st.hold(dec, st.dashes, name, dashKind(c))
		}
		return skip(dec)
	})
}

// hold keeps one named definition and skips the rest of its element.
func (st *styleSet) hold(dec *officezip.Decoder, into map[string]string, name, value string) error {
	if err := st.define(name, value); err != nil {
		return err
	}
	if name != "" && value != "" {
		into[name] = value
	}
	return skip(dec)
}

// readStyle reads one style:style or style:default-style.
func (st *styleSet) readStyle(dec *officezip.Decoder, start xml.StartElement) (*odfStyle, error) {
	s := &odfStyle{
		parent:    attr(start, nsStyle, "parent-style-name"),
		listStyle: attr(start, nsStyle, "list-style-name"),
		props:     map[string]string{},
	}
	if err := st.define(attr(start, nsStyle, "name"), s.parent, s.listStyle); err != nil {
		return nil, err
	}
	err := children(dec, func(c xml.StartElement) error {
		if c.Name.Space != nsStyle || !strings.HasSuffix(c.Name.Local, "-properties") {
			return skip(dec)
		}
		for _, a := range c.Attr {
			prefix, ok := nsPrefixes[a.Name.Space]
			key := prefix + ":" + a.Name.Local
			if !ok || !heldProperties[key] {
				continue
			}
			if _, set := s.props[key]; set {
				continue
			}
			st.bytes += int64(len(key) + len(a.Value))
			if st.bytes > MaxStyleBytes {
				return fmt.Errorf("%w: the presentation's styles hold more than %d bytes", ErrTooLarge, MaxStyleBytes)
			}
			s.props[key] = a.Value
		}
		// A presentation style keeps its outline's bullets inside its
		// graphic properties.
		return children(dec, func(cc xml.StartElement) error {
			if !is(cc, nsText, "list-style") || s.list != nil {
				return skip(dec)
			}
			ls, err := st.readListStyle(dec, "")
			s.list = ls
			return err
		})
	})
	return s, err
}

// readListStyle reads one text:list-style's levels.
func (st *styleSet) readListStyle(dec *officezip.Decoder, name string) (*odfListStyle, error) {
	if err := st.define(name); err != nil {
		return nil, err
	}
	ls := &odfListStyle{}
	err := children(dec, func(c xml.StartElement) error {
		level, err := strconv.Atoi(attr(c, nsText, "level"))
		if err != nil || level < 1 || level > listLevels {
			return skip(dec)
		}
		switch {
		case is(c, nsText, "list-level-style-bullet"), is(c, nsText, "list-level-style-image"):
			ls.levels[level-1] = listBullet
		case is(c, nsText, "list-level-style-number"):
			if attr(c, nsStyle, "num-format") != "" {
				ls.levels[level-1] = listNumbered
			}
		}
		return skip(dec)
	})
	return ls, err
}

// readPageLayout reads a style:page-layout's page size.
func (st *styleSet) readPageLayout(dec *officezip.Decoder, name string) error {
	if err := st.define(name); err != nil {
		return err
	}
	return children(dec, func(c xml.StartElement) error {
		if is(c, nsStyle, "page-layout-properties") {
			w, okW := parseCm(attr(c, nsFo, "page-width"))
			h, okH := parseCm(attr(c, nsFo, "page-height"))
			if okW && okH {
				st.pageLayouts[name] = [2]float64{w, h}
			}
		}
		return skip(dec)
	})
}

// dashKind is the .qslide dash a draw:stroke-dash draws closest to: dots
// alone, dots of two lengths, or dashes.
func dashKind(c xml.StartElement) string {
	dots1, _ := strconv.Atoi(attr(c, nsDraw, "dots1"))
	dots2, _ := strconv.Atoi(attr(c, nsDraw, "dots2"))
	len1, len2 := attr(c, nsDraw, "dots1-length"), attr(c, nsDraw, "dots2-length")
	short := func(length string) bool {
		if length == "" {
			return true
		}
		if pct, ok := parsePercent(length); ok {
			return pct <= 1
		}
		cm, ok := parseCm(length)
		return ok && cm <= 0.05
	}
	switch {
	case dots1 > 0 && dots2 > 0 && short(len1) != short(len2):
		return "dashDot"
	case short(len1) && (dots2 == 0 || short(len2)):
		return "dot"
	}
	return "dash"
}

// unquoteFamily is a font family's first name without its quotes.
func unquoteFamily(family string) string {
	first, _, _ := strings.Cut(family, ",")
	return strings.Trim(strings.TrimSpace(first), `'"`)
}

// find is the style family:name as scope sees it.
func (st *styleSet) find(scope int, family, name string) *odfStyle {
	if name == "" {
		return nil
	}
	key := family + ":" + name
	if s := st.auto[scope][key]; s != nil {
		return s
	}
	return st.common[key]
}

// parentOf is s's parent: a common style's parent is a common style.
func (st *styleSet) parentOf(scope int, family string, s *odfStyle) *odfStyle {
	if s.parent == "" {
		return nil
	}
	if s.common {
		return st.common[family+":"+s.parent]
	}
	return st.find(scope, family, s.parent)
}

// each hands fn every style refs name and their parents, nearest first,
// then the default styles of their families, until fn returns true.
func (st *styleSet) each(scope int, refs []styleRef, fn func(*odfStyle) bool) {
	for _, r := range refs {
		s := st.find(scope, r.family, r.name)
		for depth := 0; s != nil && depth < maxStyleChain; depth++ {
			if fn(s) {
				return
			}
			s = st.parentOf(scope, r.family, s)
		}
	}
	seen := make([]*odfStyle, 0, len(refs))
	for _, r := range refs {
		family := r.family
		if family == "presentation" {
			// Presentation styles have no default of their own; they
			// draw as graphic styles do.
			family = "graphic"
		}
		if d := st.defaults[family]; d != nil && !slices.Contains(seen, d) {
			seen = append(seen, d)
			if fn(d) {
				return
			}
		}
	}
}

// prop is the first value of the property key along refs' chains.
func (st *styleSet) prop(scope int, refs []styleRef, key string) (string, bool) {
	var value string
	var found bool
	st.each(scope, refs, func(s *odfStyle) bool {
		value, found = s.props[key]
		return found
	})
	return value, found
}

// listStyle is the list style name names, as scope sees it.
func (st *styleSet) listStyle(scope int, name string) *odfListStyle {
	if ls := st.autoLists[scope][name]; ls != nil {
		return ls
	}
	return st.lists[name]
}
