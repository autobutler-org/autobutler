package odputil

// cspell:ignore xlink

import "strings"

// newStyleSet is an empty style set.
func newStyleSet() *styleSet {
	return &styleSet{
		named:      map[string]*odfStyle{},
		auto:       [2]map[string]*odfStyle{{}, {}},
		lists:      [2]map[string]listStyle{{}, {}},
		namedLists: map[string]listStyle{},
		fonts:      map[string]string{},
		fillImages: map[string]string{},
		gradients:  map[string]string{},
		layouts:    map[string]pageLayout{},
	}
}

// styleKey is a style's key within its family.
func styleKey(family, name string) string { return family + "\x00" + name }

// add reads one of office:font-face-decls, office:styles,
// office:automatic-styles and office:master-styles. Automatic styles go to
// scope; everything else is the package's.
func (ss *styleSet) add(container *node, scope int) {
	automatic := container.is(nsOffice, "automatic-styles")
	for _, n := range container.children {
		switch {
		case n.is(nsStyle, "font-face"):
			if family := fontFamily(n.attr(nsSVG, "font-family")); family != "" {
				ss.fonts[n.attr(nsStyle, "name")] = family
			}
		case n.is(nsStyle, "style"), n.is(nsStyle, "default-style"):
			s := &odfStyle{
				parent:    n.attr(nsStyle, "parent-style-name"),
				listStyle: n.attr(nsStyle, "list-style-name"),
				props:     map[string]props{},
			}
			for _, p := range n.children {
				if p.name.Space != nsStyle || !strings.HasSuffix(p.name.Local, "-properties") {
					continue
				}
				values := props{}
				for _, a := range p.attrs {
					if prefix, ok := propPrefixes[a.Name.Space]; ok {
						values[prefix+":"+a.Name.Local] = a.Value
					}
				}
				s.props[p.name.Local] = values
			}
			name := n.attr(nsStyle, "name")
			if n.is(nsStyle, "default-style") {
				name = ""
			}
			key := styleKey(n.attr(nsStyle, "family"), name)
			if automatic && name != "" {
				ss.auto[scope][key] = s
			} else {
				ss.named[key] = s
			}
		case n.is(nsText, "list-style"):
			list := listStyle{}
			for _, level := range n.children {
				kind := ""
				switch {
				case level.is(nsText, "list-level-style-number"):
					kind = "numbered"
				case level.is(nsText, "list-level-style-bullet"), level.is(nsText, "list-level-style-image"):
					kind = "bullet"
				default:
					continue
				}
				if l := atoiDefault(level.attr(nsText, "level"), 1); l >= 1 && l <= 10 {
					list[l] = kind
				}
			}
			if automatic {
				ss.lists[scope][n.attr(nsStyle, "name")] = list
			} else {
				ss.namedLists[n.attr(nsStyle, "name")] = list
			}
		case n.is(nsStyle, "page-layout"):
			if p := n.child(nsStyle, "page-layout-properties"); p != nil {
				w, wok := parseCM(p.attr(nsFO, "page-width"))
				h, hok := parseCM(p.attr(nsFO, "page-height"))
				if wok && hok {
					ss.layouts[n.attr(nsStyle, "name")] = pageLayout{width: w, height: h}
				}
			}
		case n.is(nsDraw, "fill-image"):
			ss.fillImages[n.attr(nsDraw, "name")] = n.attr(nsXLink, "href")
		case n.is(nsDraw, "gradient"):
			ss.gradients[n.attr(nsDraw, "name")] = parseColor(n.attr(nsDraw, "start-color"))
		case n.is(nsStyle, "master-page"):
			m := &masterPage{
				name:      n.attr(nsStyle, "name"),
				layout:    n.attr(nsStyle, "page-layout-name"),
				pageStyle: n.attr(nsDraw, "style-name"),
			}
			for _, c := range n.children {
				if c.name.Local != "" && !c.is(nsPresentation, "notes") {
					m.shapes = append(m.shapes, c)
				}
			}
			ss.masters = append(ss.masters, m)
		}
	}
}

// find is the style a scope names, automatic first.
func (ss *styleSet) find(ref styleRef) *odfStyle {
	if ref.name == "" {
		return nil
	}
	key := styleKey(ref.family, ref.name)
	if s, ok := ss.auto[ref.scope][key]; ok {
		return s
	}
	return ss.named[key]
}

// prop looks a property up through refs, nearest first: each style and its
// parents, then each family's default style.
func (ss *styleSet) prop(refs []styleRef, kind, key string) (string, bool) {
	for _, ref := range refs {
		s := ss.find(ref)
		for depth := 0; s != nil && depth < maxStyleChain; depth++ {
			if v, ok := s.props[kind][key]; ok {
				return v, true
			}
			if s.parent == "" {
				break
			}
			s = ss.named[styleKey(ref.family, s.parent)]
		}
	}
	for _, ref := range refs {
		if s := ss.named[styleKey(ref.family, "")]; s != nil {
			if v, ok := s.props[kind][key]; ok {
				return v, true
			}
		}
	}
	return "", false
}

// get is prop without the found flag.
func (ss *styleSet) get(refs []styleRef, kind, key string) string {
	v, _ := ss.prop(refs, kind, key)
	return v
}

// listKind is whether a list style numbers or bullets level: bullet when it
// does not say.
func (ss *styleSet) listKind(scope int, name string, level int) string {
	list, ok := ss.lists[scope][name]
	if !ok {
		list = ss.namedLists[name]
	}
	if kind, ok := list[min(level, 10)]; ok {
		return kind
	}
	return "bullet"
}

// master is the master page named, or the first when none is.
func (ss *styleSet) master(name string) *masterPage {
	for _, m := range ss.masters {
		if m.name == name {
			return m
		}
	}
	if len(ss.masters) > 0 {
		return ss.masters[0]
	}
	return nil
}

// graphicRefs are the styles a drawing element is drawn with: its
// presentation style, then its graphic style.
func graphicRefs(n *node, scope int) []styleRef {
	var refs []styleRef
	if name := n.attr(nsPresentation, "style-name"); name != "" {
		refs = append(refs, styleRef{scope: scope, family: "presentation", name: name})
	}
	if name := n.attr(nsDraw, "style-name"); name != "" {
		refs = append(refs, styleRef{scope: scope, family: "graphic", name: name})
	}
	// The graphic family's default applies to every drawing element.
	return append(refs, styleRef{scope: scope, family: "graphic"})
}
