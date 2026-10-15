package odputil

// cspell:ignore xlink

import (
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/officezip"
)

// shapeKinds maps a custom shape's draw:type to the editor's shape kind.
var shapeKinds = map[string]string{
	"rectangle": "rectangle", "round-rectangle": "roundedRectangle", "ellipse": "ellipse",
	"isosceles-triangle": "triangle", "diamond": "diamond", "right-arrow": "arrow", "star5": "star",
	// Impress's flowchart process box and quad are plain rectangles.
	"flowchart-process": "rectangle", "quad-bevel": "rectangle",
}

// freeformTypes are the custom shape types a path draws.
var freeformTypes = map[string]bool{"non-primitive": true, "": true}

// roundRectangleScale is what a round-rectangle's first modifier is a share
// of: its shorter side is 21,600 units of the shape's coordinate space.
const roundRectangleScale = 21_600

// element converts one child of a draw:page, a master page or a group,
// reading it through to its end. What the editor has no model for is
// skipped with a warning.
func (s *slideReader) element(dec *officezip.Decoder, c xml.StartElement, depth int) ([]outElement, error) {
	if s.leftOut(c) {
		return nil, skip(dec)
	}
	switch c.Name.Space {
	case nsDraw:
		switch c.Name.Local {
		case "frame":
			return s.frame(dec, c)
		case "custom-shape", "rect", "ellipse", "circle", "polygon", "polyline", "path", "regular-polygon":
			return s.shape(dec, c)
		case "line", "connector":
			return s.line(dec, c)
		case "g":
			return s.group(dec, c, depth)
		case "a":
			// A link around shapes draws the shapes; the link is not
			// followed.
			var out []outElement
			err := children(dec, func(cc xml.StartElement) error {
				els, err := s.element(dec, cc, depth)
				out = append(out, els...)
				return err
			})
			return out, err
		case "measure", "caption":
			s.warn("Measurement lines and callouts are not imported.")
		case "control":
			s.warn("Form controls are not imported.")
		}
	case nsDr3d:
		if c.Name.Local == "scene" {
			s.warn("3D objects are not imported.")
		}
	}
	return nil, skip(dec)
}

// leftOut reports an element a slide does not draw: an empty placeholder,
// which is only a prompt, and on a master page every placeholder, which is
// only the slides' template.
func (s *slideReader) leftOut(c xml.StartElement) bool {
	if s.master {
		return attr(c, nsPresentation, "class") != ""
	}
	return isTrue(attr(c, nsPresentation, "placeholder"))
}

// refs are the styles an element's drawing is looked up in.
func refs(c xml.StartElement) []styleRef {
	return []styleRef{{"graphic", attr(c, nsDraw, "style-name")}, {"presentation", attr(c, nsPresentation, "style-name")}}
}

// frame converts a draw:frame by what it holds: a text box, a picture or a
// table. An embedded object, chart or plugin is skipped with a warning.
func (s *slideReader) frame(dec *officezip.Decoder, c xml.StartElement) ([]outElement, error) {
	frame := s.box(c)
	ctx := s.textContext(c)
	var out []outElement
	var images []string
	var alt string
	done := false
	err := children(dec, func(cc xml.StartElement) error {
		if done && !is(cc, nsSVG, "title") && !is(cc, nsSVG, "desc") {
			return skip(dec)
		}
		switch {
		case is(cc, nsDraw, "text-box"):
			done = true
			paragraphs, err := s.textBody(dec, ctx)
			if err != nil {
				return err
			}
			if err := s.countElement(); err != nil {
				return err
			}
			out = append(out, s.textBox(c, frame, paragraphs, "", ""))
			return nil
		case is(cc, nsDraw, "image"):
			// A frame may hold the same picture more than once, an SVG
			// and its PNG fallback; the first the editor shows is drawn.
			images = append(images, s.imageHref(cc))
			return skip(dec)
		case is(cc, nsTable, "table"):
			done = true
			table, err := s.table(dec, cc, c, frame)
			out = append(out, table...)
			return err
		case is(cc, nsDraw, "object"), is(cc, nsDraw, "object-ole"):
			done = true
			href := strings.TrimSuffix(strings.TrimPrefix(attr(cc, nsXlink, "href"), "./"), "/")
			if s.charts[href] {
				s.warn("Charts are not imported.")
			} else {
				s.warn("Embedded objects are not imported.")
			}
			return skip(dec)
		case is(cc, nsDraw, "plugin"):
			done = true
			s.warn("Video and audio are not imported.")
			return skip(dec)
		case is(cc, nsDraw, "applet"), is(cc, nsDraw, "floating-frame"):
			done = true
			s.warn("Embedded objects are not imported.")
			return skip(dec)
		case is(cc, nsSVG, "title"), is(cc, nsSVG, "desc"):
			t, err := text(dec, maxTitleBytes)
			if alt == "" || cc.Name.Local == "desc" {
				alt = strings.TrimSpace(strings.ToValidUTF8(t, ""))
			}
			return err
		}
		return skip(dec)
	})
	if err != nil || len(images) == 0 || len(out) > 0 {
		return out, err
	}
	for _, href := range images {
		ref, ok, err := s.picture(href)
		if err != nil {
			return nil, err
		}
		if ok {
			if err := s.countElement(); err != nil {
				return nil, err
			}
			return []outElement{outImage{ID: s.id(), Type: typeImage, Frame: frame, Source: ref, AltText: alt, Fit: "fill"}}, nil
		}
	}
	return nil, nil
}

// shape converts a custom shape or one of the plain shapes ODF keeps from
// before custom shapes: the shape, and a text box over it when it holds
// text, which is how the editor draws text inside a shape.
func (s *slideReader) shape(dec *officezip.Decoder, c xml.StartElement) ([]outElement, error) {
	frame := s.box(c)
	r := refs(c)
	kind := "rectangle"
	var radius *float64
	switch c.Name.Local {
	case "ellipse", "circle":
		kind = "ellipse"
	case "rect":
		if cm, ok := parseCm(attr(c, nsDraw, "corner-radius")); ok && cm > 0 {
			kind = "roundedRectangle"
			v := s.units(cm)
			radius = &v
		}
	case "polygon", "polyline", "path", "regular-polygon":
		s.warn("Freeform shapes were drawn as rectangles.")
	}
	ctx := s.textContext(c)
	var paragraphs []outParagraph
	err := children(dec, func(cc xml.StartElement) error {
		if is(cc, nsDraw, "enhanced-geometry") && c.Name.Local == "custom-shape" {
			kind, radius = s.customKind(cc, frame)
			return skip(dec)
		}
		ps, err := s.textBlock(dec, cc, ctx, nil)
		paragraphs = append(paragraphs, ps...)
		return err
	})
	if err != nil {
		return nil, err
	}
	if c.Name.Local == "custom-shape" && kind == "" {
		kind = "rectangle"
		s.warn("Freeform shapes were drawn as rectangles.")
	}
	if err := s.countElement(); err != nil {
		return nil, err
	}
	shape := outShape{
		ID: s.id(), Type: typeShape, Frame: frame, Kind: kind, Fill: s.fill(r), Stroke: s.stroke(r),
		CornerRadius: radius, Opacity: s.opacity(r),
	}
	out := []outElement{shape}
	if hasText(paragraphs) {
		if err := s.countElement(); err != nil {
			return nil, err
		}
		out = append(out, s.textBox(c, frame, paragraphs, "middle", "fixed"))
	}
	return out, nil
}

// customKind is the editor's kind for a custom shape's geometry, and its
// corner radius when it is a rounded rectangle; "" for a freeform path.
func (s *slideReader) customKind(geometry xml.StartElement, frame outFrame) (string, *float64) {
	typ := attr(geometry, nsDraw, "type")
	if freeformTypes[typ] {
		return "", nil
	}
	kind, ok := shapeKinds[typ]
	if !ok {
		s.warn(fmt.Sprintf("Shapes of a kind the editor does not have (%s) were drawn as rectangles.", sanitize(typ)))
		return "rectangle", nil
	}
	if kind != "roundedRectangle" {
		return kind, nil
	}
	first, _, _ := strings.Cut(strings.TrimSpace(attr(geometry, nsDraw, "modifiers")), " ")
	n, err := strconv.ParseFloat(first, 64)
	if err != nil || math.IsNaN(n) || n < 0 {
		return kind, nil
	}
	radius := round(math.Min(n/roundRectangleScale, 0.5) * math.Min(frame.Width, frame.Height))
	return kind, &radius
}

// sanitize keeps a name from the file to what a warning may show.
func sanitize(name string) string {
	var b strings.Builder
	for _, r := range name {
		if b.Len() >= 40 {
			break
		}
		if r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "unknown"
	}
	return b.String()
}

// line converts a draw:line or draw:connector, drawn straight across its
// frame's diagonal; flipped when it runs bottom left to top right.
func (s *slideReader) line(dec *officezip.Decoder, c xml.StartElement) ([]outElement, error) {
	if err := skip(dec); err != nil {
		return nil, err
	}
	r := refs(c)
	stroke := s.stroke(r)
	if stroke == nil {
		return nil, nil
	}
	if err := s.countElement(); err != nil {
		return nil, err
	}
	if is(c, nsDraw, "connector") && attr(c, nsDraw, "type") != "line" {
		s.warn("Bent and curved connectors were drawn as straight lines.")
	}
	x1, y1, x2, y2 := endpoints(c)
	line := outLine{
		ID: s.id(), Type: typeLine, Stroke: *stroke, Opacity: s.opacity(r),
		Frame: outFrame{
			X: s.units(math.Min(x1, x2)), Y: s.units(math.Min(y1, y2)),
			Width: s.units(math.Abs(x2 - x1)), Height: s.units(math.Abs(y2 - y1)),
		},
		Flipped: (x2-x1)*(y2-y1) < 0,
	}
	start, _ := s.styles.prop(s.scope, r, "draw:marker-start")
	end, _ := s.styles.prop(s.scope, r, "draw:marker-end")
	line.StartCap, line.EndCap = markerCap(start), markerCap(end)
	// The editor's line starts at its frame's left, or its top when it is
	// upright.
	if x1 > x2 || (x1 == x2 && y1 > y2) {
		line.StartCap, line.EndCap = line.EndCap, line.StartCap
	}
	return []outElement{line}, nil
}

// markerCap is the cap a line end with the named marker imports as.
func markerCap(marker string) string {
	if strings.TrimSpace(marker) == "" {
		return ""
	}
	return "arrow"
}

// group converts a draw:g. ODF places a group's children on the page, so
// the group's frame is the box around them, and they are moved into it.
func (s *slideReader) group(dec *officezip.Decoder, c xml.StartElement, depth int) ([]outElement, error) {
	if depth >= MaxGroupDepth {
		return nil, fmt.Errorf("%w: groups nest more than %d deep", ErrTooLarge, MaxGroupDepth)
	}
	var kids []outElement
	err := children(dec, func(cc xml.StartElement) error {
		els, err := s.element(dec, cc, depth+1)
		kids = append(kids, els...)
		return err
	})
	if err != nil || len(kids) == 0 {
		return nil, err
	}
	if err := s.countElement(); err != nil {
		return nil, err
	}
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, k := range kids {
		f := frameOf(k)
		minX, minY = math.Min(minX, f.X), math.Min(minY, f.Y)
		maxX, maxY = math.Max(maxX, f.X+f.Width), math.Max(maxY, f.Y+f.Height)
	}
	for i, k := range kids {
		kids[i] = moved(k, -minX, -minY)
	}
	return []outElement{outGroup{
		ID: s.id(), Type: typeGroup, Children: kids,
		Frame: outFrame{X: round(minX), Y: round(minY), Width: round(maxX - minX), Height: round(maxY - minY)},
	}}, nil
}

// frameOf is an element's frame.
func frameOf(el outElement) outFrame {
	switch e := el.(type) {
	case outText:
		return e.Frame
	case outShape:
		return e.Frame
	case outImage:
		return e.Frame
	case outLine:
		return e.Frame
	case outGroup:
		return e.Frame
	case outTable:
		return e.Frame
	}
	return outFrame{}
}

// moved is an element with its frame moved by dx, dy.
func moved(el outElement, dx, dy float64) outElement {
	move := func(f outFrame) outFrame {
		f.X, f.Y = round(f.X+dx), round(f.Y+dy)
		return f
	}
	switch e := el.(type) {
	case outText:
		e.Frame = move(e.Frame)
		return e
	case outShape:
		e.Frame = move(e.Frame)
		return e
	case outImage:
		e.Frame = move(e.Frame)
		return e
	case outLine:
		e.Frame = move(e.Frame)
		return e
	case outGroup:
		e.Frame = move(e.Frame)
		return e
	case outTable:
		e.Frame = move(e.Frame)
		return e
	}
	return el
}

// fill is the color an element is filled with, "" for none. A fill no
// style sets is none.
func (s *slideReader) fill(r []styleRef) string {
	fill, _ := s.styles.prop(s.scope, r, "draw:fill")
	switch fill {
	case "solid", "gradient", "hatch":
		return s.fillColor(r, fill)
	case "bitmap":
		s.warn("Picture fills of shapes are not imported.")
	}
	return ""
}

// stroke is an element's outline, nil for none. An outline no style sets
// is none; a hairline, width 0, is drawn one unit wide.
func (s *slideReader) stroke(r []styleRef) *outStroke {
	kind, _ := s.styles.prop(s.scope, r, "draw:stroke")
	if kind != "solid" && kind != "dash" {
		return nil
	}
	color, _ := s.styles.prop(s.scope, r, "svg:stroke-color")
	stroke := &outStroke{Color: parseColor(color), Width: 1}
	if stroke.Color == "" {
		stroke.Color = "#000000"
	}
	if v, ok := s.styles.prop(s.scope, r, "svg:stroke-width"); ok {
		if cm, ok := parseCm(v); ok && cm > 0 {
			stroke.Width = math.Max(s.units(cm), 0.01)
		}
	}
	if kind == "dash" {
		name, _ := s.styles.prop(s.scope, r, "draw:stroke-dash")
		stroke.Dash = s.styles.dashes[name]
		if stroke.Dash == "" {
			stroke.Dash = "dash"
		}
	}
	return stroke
}

// opacity is an element's opacity, nil when opaque.
func (s *slideReader) opacity(r []styleRef) *float64 {
	v, ok := s.styles.prop(s.scope, r, "draw:opacity")
	if !ok {
		return nil
	}
	o, ok := parsePercent(v)
	if !ok || o >= 1 {
		return nil
	}
	o = round(o)
	return &o
}
