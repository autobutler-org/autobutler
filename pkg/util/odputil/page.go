package odputil

// cspell:ignore crossfade smil

import (
	"encoding/xml"
	"math"
	"strconv"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/officezip"
)

// Transition durations, in milliseconds, as quark_slides keeps them.
const (
	minTransitionMs     = 200
	maxTransitionMs     = 2000
	defaultTransitionMs = 500
)

// The transition speeds Impress names, in milliseconds; medium is the
// editor's default and is left out.
var transitionSpeeds = map[string]int{"fast": 300, "slow": 1000}

// smilDirections maps a transition's subtype to the way the slides travel.
var smilDirections = map[string]string{
	// pushWipe: where the new slide comes from.
	"fromLeft": "right", "fromRight": "left", "fromTop": "down", "fromBottom": "up",
	// barWipe: which way the edge moves.
	"leftToRight": "right", "topToBottom": "down",
}

// reverseDirection is the way the slides travel when a transition runs in
// reverse.
var reverseDirection = map[string]string{"left": "right", "right": "left", "up": "down", "down": "up"}

// background reads a drawing-page style's fill as a slide background. set
// is false when the style sets no fill of its own, so the master's shows.
func (s *slideReader) background(styleName string) (*outBackground, bool, error) {
	if styleName == "" {
		return nil, false, nil
	}
	refs := []styleRef{{"drawing-page", styleName}}
	fill, _ := s.styles.prop(s.scope, refs, "draw:fill")
	switch fill {
	case "bitmap":
		name, _ := s.styles.prop(s.scope, refs, "draw:fill-image-name")
		ref, ok, err := s.picture(s.styles.fillImages[name])
		if err != nil || !ok {
			return nil, false, err
		}
		return &outBackground{Image: ref}, true, nil
	case "solid", "gradient", "hatch":
		if c := s.fillColor(refs, fill); c != "" {
			return &outBackground{Color: c}, true, nil
		}
	}
	return nil, false, nil
}

// fillColor is the one color a solid, gradient or hatch fill is drawn in.
// A gradient takes its start color and a hatch its lines' color, with a
// warning.
func (s *slideReader) fillColor(refs []styleRef, fill string) string {
	color, _ := s.styles.prop(s.scope, refs, "draw:fill-color")
	switch fill {
	case "gradient", "hatch":
		key, defs := "draw:fill-gradient-name", s.styles.gradients
		if fill == "hatch" {
			key, defs = "draw:fill-hatch-name", s.styles.hatches
		}
		if name, _ := s.styles.prop(s.scope, refs, key); defs[name] != "" {
			color = defs[name]
		}
		s.warn("Gradient and hatched fills were drawn in a single color.")
	}
	return parseColor(color)
}

// transition reads a drawing-page style's transition: the effect the
// editor has that matches it, or nil, with a warning when there is an
// effect and the editor has no match for it.
func (s *slideReader) transition(refs []styleRef) *outTransition {
	smilType, _ := s.styles.prop(s.scope, refs, "smil:type")
	legacy, _ := s.styles.prop(s.scope, refs, "presentation:transition-style")
	if smilType == "" && (legacy == "" || legacy == "none") {
		return nil
	}
	subtype, _ := s.styles.prop(s.scope, refs, "smil:subtype")
	var t outTransition
	switch {
	case smilType == "fade", smilType == "" && strings.HasPrefix(legacy, "fade-"):
		t.Kind = "fade"
	case smilType == "pushWipe":
		t.Kind, t.Direction = "push", smilDirections[subtype]
	case smilType == "barWipe":
		t.Kind, t.Direction = "wipe", smilDirections[subtype]
	case smilType == "zoom":
		t.Kind = "zoom"
	default:
		s.warn("This slide's transition has no match in the editor, so it was left out.")
		return nil
	}
	if t.Direction == "" && (t.Kind == "push" || t.Kind == "wipe") {
		t.Direction = "left"
	}
	if dir, _ := s.styles.prop(s.scope, refs, "smil:direction"); dir == "reverse" && t.Direction != "" {
		t.Direction = reverseDirection[t.Direction]
	}
	if t.Direction == "left" {
		// The codec's default, which it leaves out.
		t.Direction = ""
	}
	speed, _ := s.styles.prop(s.scope, refs, "presentation:transition-speed")
	t.Duration = transitionSpeeds[speed]
	if s.transitionMs > 0 {
		t.Duration = s.transitionMs
	}
	if t.Duration == defaultTransitionMs {
		t.Duration = 0
	}
	return &t
}

// parseDuration reads a SMIL clock value, "2s", "500ms" or "0.5s", as
// milliseconds within the bounds the editor keeps; 0 when it is not one.
func parseDuration(v string) int {
	v = strings.TrimSpace(v)
	unit := 1000.0
	if n, ok := strings.CutSuffix(v, "ms"); ok {
		v, unit = n, 1
	} else if n, ok := strings.CutSuffix(v, "s"); ok {
		v = n
	} else {
		return 0
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil || math.IsNaN(n) || n <= 0 {
		return 0
	}
	return int(math.Max(minTransitionMs, math.Min(maxTransitionMs, math.Round(n*unit))))
}

// notes reads a presentation:notes: the text of its notes frame, a line
// per paragraph.
func (s *slideReader) notes(dec *officezip.Decoder) (string, error) {
	var lines []string
	err := children(dec, func(c xml.StartElement) error {
		if !is(c, nsDraw, "frame") || attr(c, nsPresentation, "class") != "notes" {
			return skip(dec)
		}
		ctx := s.textContext(c)
		return children(dec, func(box xml.StartElement) error {
			if !is(box, nsDraw, "text-box") {
				return skip(dec)
			}
			paragraphs, err := s.textBody(dec, ctx)
			for _, p := range paragraphs {
				var line strings.Builder
				for _, run := range p.Runs {
					line.WriteString(run.Text)
				}
				lines = append(lines, line.String())
			}
			return err
		})
	})
	return strings.TrimRight(strings.Join(lines, "\n"), "\n"), err
}

// reID copies a master page's element for one slide, under fresh ids, and
// charges it to the deck's element budget as the slide's own would be.
func (s *slideReader) reID(el outElement) (outElement, error) {
	if err := s.countElement(); err != nil {
		return nil, err
	}
	switch e := el.(type) {
	case outText:
		e.ID = s.id()
		return e, nil
	case outShape:
		e.ID = s.id()
		return e, nil
	case outImage:
		e.ID = s.id()
		return e, nil
	case outLine:
		e.ID = s.id()
		return e, nil
	case outTable:
		for range len(e.Rows)*len(e.Columns) - 1 {
			if err := s.countElement(); err != nil {
				return nil, err
			}
		}
		e.ID = s.id()
		return e, nil
	case outGroup:
		kids := make([]outElement, 0, len(e.Children))
		for _, child := range e.Children {
			copied, err := s.reID(child)
			if err != nil {
				return nil, err
			}
			kids = append(kids, copied)
		}
		e.ID, e.Children = s.id(), kids
		return e, nil
	}
	return el, nil
}
