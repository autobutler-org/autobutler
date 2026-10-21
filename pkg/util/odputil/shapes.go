package odputil

// cspell:ignore quadrat xlink

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// odfKinds maps a custom shape's draw:type to the shape kind it imports as.
var odfKinds = map[string]string{
	"rectangle":           "rectangle",
	"quadrat":             "rectangle",
	"flowchart-process":   "rectangle",
	"round-rectangle":     "roundedRectangle",
	"round-quadrat":       "roundedRectangle",
	"ellipse":             "ellipse",
	"circle":              "ellipse",
	"flowchart-connector": "ellipse",
	"isosceles-triangle":  "triangle",
	"diamond":             "diamond",
	"flowchart-decision":  "diamond",
	"right-arrow":         "arrow",
	"star5":               "star",
}

// odfPresets maps a custom shape's draw:type to the preset geometry the
// editor draws it with, as a path shape.
var odfPresets = map[string]string{
	"right-triangle":  "rtTriangle",
	"flowchart-delay": "flowChartDelay",
}

// ooxmlKinds maps the PowerPoint preset a draw:type of ooxml-<preset> names
// — how LibreOffice keeps a shape it read from a .pptx — to its kind.
var ooxmlKinds = map[string]string{
	"rect":       "rectangle",
	"roundRect":  "roundedRectangle",
	"ellipse":    "ellipse",
	"triangle":   "triangle",
	"diamond":    "diamond",
	"rightArrow": "arrow",
	"star5":      "star",
}

// ooxmlPresets are the PowerPoint presets the editor draws as a path shape.
// Keep this set in step with pptxutil's pathPresets.
var ooxmlPresets = map[string]bool{
	"diagStripe":     true,
	"flowChartDelay": true,
	"round2SameRect": true,
	"rtTriangle":     true,
	"trapezoid":      true,
}

// convertChildren converts a slide's, a master's or a group's elements in
// stacking order.
func (s *slideReader) convertChildren(nodes []*node, depth int) ([]outElement, error) {
	if depth > MaxGroupDepth {
		return nil, fmt.Errorf("%w: groups nest more than %d deep", ErrTooLarge, MaxGroupDepth)
	}
	var out []outElement
	for _, n := range nodes {
		if n.name.Local == "" || n.name.Space != nsDraw {
			continue
		}
		if d := n.attr(nsDraw, "display"); d == "none" || d == "printer" {
			continue
		}
		var converted []outElement
		var err error
		switch n.name.Local {
		case "frame":
			converted, err = s.convertFrame(n)
		case "custom-shape", "rect", "ellipse", "circle", "polygon", "polyline", "regular-polygon":
			converted, err = s.convertShape(n)
		case "line", "connector":
			converted, err = s.convertLine(n)
		case "g":
			converted, err = s.convertGroup(n, depth)
		case "a":
			converted, err = s.convertChildren(n.children, depth)
		case "path":
			s.warn("Freeform drawings were left out.")
		case "control":
			s.warn("Form controls were left out.")
		default:
			s.warn(fmt.Sprintf("Drawings of a kind the editor does not have (%s) were left out.", n.name.Local))
		}
		if err != nil {
			return nil, err
		}
		out = append(out, converted...)
	}
	return out, nil
}

// convertFrame converts a draw:frame by what it holds: a text box, a
// picture, a table. A chart, an embedded object or a video is left out.
func (s *slideReader) convertFrame(n *node) ([]outElement, error) {
	class := n.attr(nsPresentation, "class")
	// A master's placeholders are only the slides' templates, and an empty
	// placeholder on a slide is only a prompt.
	if (s.master && class != "") || n.attr(nsPresentation, "placeholder") == "true" {
		return nil, nil
	}
	frame, ok := s.geometry(n, class)
	if !ok {
		return nil, nil
	}
	refs := graphicRefs(n, s.scope)
	var box, table *node
	var images []string
	for _, c := range n.children {
		switch {
		case c.is(nsDraw, "object"), c.is(nsDraw, "object-ole"), c.is(nsDraw, "applet"), c.is(nsDraw, "floating-frame"):
			s.warn("Charts and embedded objects were left out.")
			return nil, nil
		case c.is(nsDraw, "plugin"):
			s.warn("Video and audio were left out.")
			return nil, nil
		case c.is(nsDraw, "text-box"):
			box = c
		case c.is(nsTable, "table"):
			table = c
		case c.is(nsDraw, "image"):
			href := c.attr(nsXLink, "href")
			if href == "" && c.child(nsOffice, "binary-data") != nil {
				s.warn("Pictures saved inside the slide's XML were left out.")
				continue
			}
			images = append(images, href)
		}
	}
	switch {
	case table != nil:
		return s.convertTable(table, frame, refs)
	case len(images) > 0:
		ref, ok, err := s.pictures(images)
		if err != nil || !ok {
			return nil, err
		}
		if err := s.countElement(1); err != nil {
			return nil, err
		}
		alt := ""
		if d := n.child(nsSVG, "desc"); d != nil {
			alt = strings.TrimSpace(d.innerText())
		}
		if t := n.child(nsSVG, "title"); alt == "" && t != nil {
			alt = strings.TrimSpace(t.innerText())
		}
		return []outElement{&outImage{
			ID: s.id(), Type: typeImage, Frame: frame, Source: ref, AltText: alt, Fit: "fill",
			Shadow: s.shadow(refs),
		}}, nil
	case box != nil:
		return s.textFrame(box, frame, refs, class != "")
	}
	return nil, nil
}

// textFrame converts a frame's text box, over a rectangle when the frame is
// filled or outlined, which is how the editor draws a text box with a
// background.
func (s *slideReader) textFrame(box *node, frame outFrame, refs []styleRef, placeholder bool) ([]outElement, error) {
	paragraphs, err := s.paragraphs(box, refs)
	if err != nil {
		return nil, err
	}
	var out []outElement
	fill, stroke := s.fill(refs, false), s.stroke(refs, false)
	if fill != "" || stroke != nil {
		if err := s.countElement(1); err != nil {
			return nil, err
		}
		out = append(out, &outShape{
			ID: s.id(), Type: typeShape, Frame: frame, Kind: "rectangle", Fill: fill, Stroke: stroke, Shadow: s.shadow(refs),
		})
	}
	// A placeholder with no text is only a prompt; an empty text box of the
	// slide's own is kept.
	if visibleText(paragraphs) || !placeholder {
		if err := s.countElement(1); err != nil {
			return nil, err
		}
		out = append(out, s.textBox(frame, paragraphs, refs, ""))
	}
	return out, nil
}

// textBox is a text box over frame, anchored as its style says, or at
// anchor when the style says nothing.
func (s *slideReader) textBox(frame outFrame, paragraphs []outParagraph, refs []styleRef, anchor string) *outText {
	if v, ok := s.styles.prop(refs, "graphic-properties", "draw:textarea-vertical-align"); ok {
		anchor = map[string]string{"middle": "middle", "bottom": "bottom"}[v]
	}
	autoFit := ""
	switch {
	case s.styles.get(refs, "graphic-properties", "style:shrink-to-fit") == "true",
		strings.Contains(s.styles.get(refs, "graphic-properties", "draw:fit-to-size"), "shrink"),
		s.styles.get(refs, "graphic-properties", "draw:fit-to-size") == "true":
		autoFit = "shrink"
	case s.styles.get(refs, "graphic-properties", "draw:auto-grow-height") == "false":
		autoFit = "fixed"
	}
	if paragraphs == nil {
		paragraphs = []outParagraph{}
	}
	return &outText{ID: s.id(), Type: typeText, Frame: frame, Paragraphs: paragraphs, Anchor: anchor, AutoFit: autoFit}
}

// convertShape converts a drawn shape, with a text box over it when it holds
// text, which is how the editor draws text inside a shape.
func (s *slideReader) convertShape(n *node) ([]outElement, error) {
	frame, ok := s.geometry(n, "")
	if !ok {
		return nil, nil
	}
	if err := s.countElement(1); err != nil {
		return nil, err
	}
	refs := graphicRefs(n, s.scope)
	shape := &outShape{
		ID: s.id(), Type: typeShape, Kind: "rectangle",
		Fill: s.fill(refs, true), Stroke: s.stroke(refs, true), Shadow: s.shadow(refs),
	}
	s.shapeKind(n, &frame, shape)
	shape.Frame = frame
	out := []outElement{shape}
	paragraphs, err := s.paragraphs(n, refs)
	if err != nil {
		return nil, err
	}
	if visibleText(paragraphs) {
		if err := s.countElement(1); err != nil {
			return nil, err
		}
		out = append(out, s.textBox(frame, paragraphs, refs, "middle"))
	}
	return out, nil
}

// shapeKind sets the kind the shape draws as, and what that kind needs: a
// corner radius, a preset, contours. A kind the editor does not have is a
// rectangle, with a warning.
func (s *slideReader) shapeKind(n *node, frame *outFrame, shape *outShape) {
	side := math.Min(frame.Width, frame.Height)
	switch n.name.Local {
	case "rect":
		if r, ok := s.length(n.attr(nsDraw, "corner-radius")); ok && r > 0 {
			radius := round(math.Min(r, side/2))
			shape.Kind, shape.CornerRadius = "roundedRectangle", &radius
		}
	case "ellipse", "circle":
		shape.Kind = "ellipse"
		if k := n.attr(nsDraw, "kind"); k != "" && k != "full" {
			s.warn("Arcs and pie slices were drawn as whole ellipses.")
		}
	case "polygon", "polyline":
		contour, ok := polygonContour(n.attr(nsDraw, "points"), n.attr(nsSVG, "viewBox"), n.name.Local == "polygon")
		if !ok {
			s.warn("Freeform shapes that could not be read were drawn as rectangles.")
			return
		}
		shape.Kind, shape.Contours = "path", []shapeContour{contour}
	case "regular-polygon":
		corners := atoiDefault(n.attr(nsDraw, "corners"), 0)
		concave := n.attr(nsDraw, "concave") == "true"
		switch {
		case concave && corners == 5:
			shape.Kind = "star"
		case !concave && corners >= 3 && corners <= 100:
			shape.Kind, shape.Contours = "path", []shapeContour{regularPolygon(corners)}
		default:
			s.warn("Stars of other than five points were drawn as rectangles.")
		}
	case "custom-shape":
		s.customShapeKind(n, side, frame, shape)
	}
}

// customShapeKind maps a draw:custom-shape's enhanced geometry.
func (s *slideReader) customShapeKind(n *node, side float64, frame *outFrame, shape *outShape) {
	geometry := n.child(nsDraw, "enhanced-geometry")
	if geometry == nil {
		return
	}
	kind, adjustScale, defaultAdjust := "", 21_600.0, 3_600.0
	t := geometry.attr(nsDraw, "type")
	if preset, ok := strings.CutPrefix(t, "ooxml-"); ok {
		kind, adjustScale, defaultAdjust = ooxmlKinds[preset], 100_000, 16_667
		if kind == "" && ooxmlPresets[preset] {
			shape.Kind, shape.Preset = "path", preset
			return
		}
	} else {
		kind = odfKinds[t]
		if preset, ok := odfPresets[t]; ok {
			shape.Kind, shape.Preset = "path", preset
			return
		}
	}
	if kind == "" {
		name := t
		if name == "" || name == "non-primitive" {
			name = "freeform"
		}
		s.warn(fmt.Sprintf("Shapes of a kind the editor does not have (%s) were drawn as rectangles.", name))
		return
	}
	shape.Kind = kind
	if kind == "roundedRectangle" {
		adjust := defaultAdjust
		if fields := strings.Fields(geometry.attr(nsDraw, "modifiers")); len(fields) > 0 {
			if v, err := strconv.ParseFloat(fields[0], 64); err == nil && v >= 0 {
				adjust = v
			}
		}
		radius := round(math.Min(adjust/adjustScale*side, side/2))
		shape.CornerRadius = &radius
	}
	// A mirrored triangle or star points down, and a mirrored arrow left:
	// the same as turning them half a turn. The other kinds are symmetric.
	mirrorH := geometry.attr(nsDraw, "mirror-horizontal") == "true"
	mirrorV := geometry.attr(nsDraw, "mirror-vertical") == "true"
	if (mirrorH && kind == "arrow") || (mirrorV && (kind == "triangle" || kind == "star")) {
		frame.Rotation = round(math.Mod(frame.Rotation+180, 360))
	}
}

// polygonContour reads draw:points in the coordinates of svg:viewBox as one
// contour in the frame's own space, closed for a polygon.
func polygonContour(points, viewBox string, closed bool) (shapeContour, bool) {
	box := strings.Fields(strings.ReplaceAll(viewBox, ",", " "))
	if len(box) != 4 {
		return shapeContour{}, false
	}
	var vb [4]float64
	for i, f := range box {
		v, err := strconv.ParseFloat(f, 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			return shapeContour{}, false
		}
		vb[i] = v
	}
	if vb[2] <= 0 || vb[3] <= 0 {
		return shapeContour{}, false
	}
	pairs := strings.Fields(points)
	if len(pairs) < 2 || len(pairs) > maxPolygonPoints {
		return shapeContour{}, false
	}
	contour := shapeContour{Commands: make([]pathCommand, 0, len(pairs)+1)}
	for i, pair := range pairs {
		xs, ys, ok := strings.Cut(pair, ",")
		x, errX := strconv.ParseFloat(xs, 64)
		y, errY := strconv.ParseFloat(ys, 64)
		if !ok || errX != nil || errY != nil || math.IsNaN(x+y) || math.IsInf(x+y, 0) {
			return shapeContour{}, false
		}
		op := "line"
		if i == 0 {
			op = "move"
		}
		contour.Commands = append(contour.Commands, pathCommand{
			Op: op, X: roundFraction((x - vb[0]) / vb[2]), Y: roundFraction((y - vb[1]) / vb[3]),
		})
	}
	if closed {
		contour.Commands = append(contour.Commands, pathCommand{Op: "close"})
	} else {
		noFill := false
		contour.Fill = &noFill
	}
	return contour, true
}

// regularPolygon is a polygon of corners corners inscribed in the frame,
// the first at its top center.
func regularPolygon(corners int) shapeContour {
	contour := shapeContour{Commands: make([]pathCommand, 0, corners+1)}
	for i := range corners {
		angle := -math.Pi/2 + 2*math.Pi*float64(i)/float64(corners)
		op := "line"
		if i == 0 {
			op = "move"
		}
		contour.Commands = append(contour.Commands, pathCommand{
			Op: op, X: roundFraction(0.5 + 0.5*math.Cos(angle)), Y: roundFraction(0.5 + 0.5*math.Sin(angle)),
		})
	}
	return shapeContour{Commands: append(contour.Commands, pathCommand{Op: "close"})}
}

// roundFraction keeps four decimals of a point in the frame's own space.
func roundFraction(v float64) float64 {
	r := math.Round(v*10_000) / 10_000
	if r == 0 {
		return 0
	}
	return r
}

// convertLine converts a draw:line or draw:connector, which runs from
// svg:x1, svg:y1 to svg:x2, svg:y2.
func (s *slideReader) convertLine(n *node) ([]outElement, error) {
	x1, ok1 := s.length(n.attr(nsSVG, "x1"))
	y1, ok2 := s.length(n.attr(nsSVG, "y1"))
	x2, ok3 := s.length(n.attr(nsSVG, "x2"))
	y2, ok4 := s.length(n.attr(nsSVG, "y2"))
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return nil, nil
	}
	refs := graphicRefs(n, s.scope)
	stroke := s.stroke(refs, true)
	if stroke == nil {
		return nil, nil
	}
	if n.is(nsDraw, "connector") && n.attr(nsDraw, "type") != "line" {
		s.warn("Bent and curved connectors were drawn as straight lines.")
	}
	if err := s.countElement(1); err != nil {
		return nil, err
	}
	line := &outLine{
		ID: s.id(), Type: typeLine, Stroke: *stroke,
		Frame:   outFrame{X: round(math.Min(x1, x2)), Y: round(math.Min(y1, y2)), Width: round(math.Abs(x2 - x1)), Height: round(math.Abs(y2 - y1))},
		Flipped: (x2-x1)*(y2-y1) < 0,
	}
	if s.styles.get(refs, "graphic-properties", "draw:marker-start") != "" {
		line.StartCap = "arrow"
	}
	if s.styles.get(refs, "graphic-properties", "draw:marker-end") != "" {
		line.EndCap = "arrow"
	}
	// The editor's line starts at its left end, or its top end when it is
	// vertical.
	if x1 > x2 || (x1 == x2 && y1 > y2) {
		line.StartCap, line.EndCap = line.EndCap, line.StartCap
	}
	return []outElement{line}, nil
}

// convertGroup converts a draw:g, its children moved into the group's own
// frame: the box around them.
func (s *slideReader) convertGroup(n *node, depth int) ([]outElement, error) {
	children, err := s.convertChildren(n.children, depth+1)
	if err != nil || len(children) == 0 {
		return nil, err
	}
	if err := s.countElement(1); err != nil {
		return nil, err
	}
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, c := range children {
		f := c.box()
		minX, minY = math.Min(minX, f.X), math.Min(minY, f.Y)
		maxX, maxY = math.Max(maxX, f.X+f.Width), math.Max(maxY, f.Y+f.Height)
	}
	for _, c := range children {
		f := c.box()
		f.X, f.Y = round(f.X-minX), round(f.Y-minY)
	}
	return []outElement{&outGroup{
		ID: s.id(), Type: typeGroup, Children: children,
		Frame: outFrame{X: round(minX), Y: round(minY), Width: round(maxX - minX), Height: round(maxY - minY)},
	}}, nil
}

// geometry is an element's frame: svg:x, svg:y, svg:width and svg:height,
// turned and moved by draw:transform. A slide's placeholder that says
// nothing of where it goes takes its master's placeholder's place.
func (s *slideReader) geometry(n *node, class string) (outFrame, bool) {
	w, wok := s.length(n.attr(nsSVG, "width"))
	h, hok := s.length(n.attr(nsSVG, "height"))
	if !wok || !hok {
		if class == "" || s.master || s.page == nil {
			return outFrame{}, false
		}
		for _, m := range s.page.shapes {
			if m.is(nsDraw, "frame") && m.attr(nsPresentation, "class") == class {
				return s.geometry(m, "")
			}
		}
		return outFrame{}, false
	}
	w, h = math.Max(w, 0), math.Max(h, 0)
	x, _ := s.length(n.attr(nsSVG, "x"))
	y, _ := s.length(n.attr(nsSVG, "y"))
	cx, cy, angle := x+w/2, y+h/2, 0.0
	if tr := n.attr(nsDraw, "transform"); tr != "" {
		var skewed bool
		cx, cy, angle, skewed = s.applyTransform(tr, cx, cy)
		if skewed {
			s.warn("Skewed shapes were drawn without their skew.")
		}
	}
	rotation := math.Mod(-angle*180/math.Pi, 360)
	if rotation < 0 {
		rotation += 360
	}
	rotation = round(rotation)
	if rotation >= 360 {
		rotation = 0
	}
	return outFrame{X: round(cx - w/2), Y: round(cy - h/2), Width: round(w), Height: round(h), Rotation: rotation}, true
}

// applyTransform moves the point x, y — an element's center — through
// draw:transform's operations in the order they are written, and returns
// where it lands and how far the element turned, in radians
// counterclockwise. A skew is left out and reported.
func (s *slideReader) applyTransform(tr string, x, y float64) (float64, float64, float64, bool) {
	angle, skewed := 0.0, false
	rest := tr
	for {
		open := strings.IndexByte(rest, '(')
		end := strings.IndexByte(rest, ')')
		if open < 0 || end < open {
			break
		}
		op := strings.TrimSpace(strings.Trim(rest[:open], " ,"))
		args := strings.Fields(strings.ReplaceAll(rest[open+1:end], ",", " "))
		rest = rest[end+1:]
		num := func(i int) float64 {
			if i >= len(args) {
				return 0
			}
			v, err := strconv.ParseFloat(args[i], 64)
			if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
				return 0
			}
			return v
		}
		lengthArg := func(i int) float64 {
			if i >= len(args) {
				return 0
			}
			if v, ok := s.length(args[i]); ok {
				return v
			}
			// A bare number is in hundredths of a millimeter.
			return num(i) * s.unit / 1000
		}
		switch op {
		case "rotate":
			a := num(0)
			x, y = x*math.Cos(a)+y*math.Sin(a), -x*math.Sin(a)+y*math.Cos(a)
			angle += a
		case "translate":
			x, y = x+lengthArg(0), y+lengthArg(1)
		case "scale":
			sx, sy := num(0), num(1)
			if len(args) < 2 {
				sy = sx
			}
			x, y = x*sx, y*sy
		case "matrix":
			a, b, c, d := num(0), num(1), num(2), num(3)
			x, y = a*x+c*y+lengthArg(4), b*x+d*y+lengthArg(5)
			angle -= math.Atan2(b, a)
		case "skewX", "skewY":
			skewed = true
		}
	}
	return x, y, angle, skewed
}

// fill is the fill a style chain sets, "" for none. implicit is whether a
// fill color alone, with no draw:fill, fills: a shape's does, a frame's
// does not.
func (s *slideReader) fill(refs []styleRef, implicit bool) string {
	get := func(key string) (string, bool) { return s.styles.prop(refs, "graphic-properties", key) }
	kind, set := get("draw:fill")
	color, _ := get("draw:fill-color")
	c := parseColor(color)
	switch {
	case kind == "solid", !set && implicit:
	case kind == "gradient":
		s.warn("Gradient fills were drawn in one color.")
		name, _ := get("draw:fill-gradient-name")
		if g := s.styles.gradients[name]; g != "" {
			c = g
		}
	case kind == "hatch":
		s.warn("Pattern fills were drawn in one color.")
	case kind == "bitmap":
		s.warn("Picture fills on shapes were left out.")
		return ""
	default:
		return ""
	}
	if opacity, ok := get("draw:opacity"); ok {
		if v, ok := parsePercent(opacity); ok {
			return withAlpha(c, v)
		}
	}
	return c
}

// stroke is the outline a style chain sets, or nil for none. implicit is
// whether a stroke color alone, with no draw:stroke, draws one.
func (s *slideReader) stroke(refs []styleRef, implicit bool) *outStroke {
	get := func(key string) (string, bool) { return s.styles.prop(refs, "graphic-properties", key) }
	kind, set := get("draw:stroke")
	colorValue, colorSet := get("svg:stroke-color")
	switch {
	case kind == "solid", kind == "dash":
	case !set && implicit && colorSet:
	default:
		return nil
	}
	c := parseColor(colorValue)
	if c == "" {
		c = "#000000"
	}
	if opacity, ok := get("svg:stroke-opacity"); ok {
		if v, ok := parsePercent(opacity); ok {
			c = withAlpha(c, v)
		}
	}
	width := 1.0
	if w, ok := get("svg:stroke-width"); ok {
		if v, ok := s.length(w); ok && v > 0 {
			width = round(math.Max(v, 0.01))
		}
	}
	stroke := &outStroke{Color: c, Width: width}
	if kind == "dash" {
		stroke.Dash = "dash"
	}
	return stroke
}

// shadow is the drop shadow a style chain sets, or nil.
func (s *slideReader) shadow(refs []styleRef) *outShadow {
	get := func(key string) string { return s.styles.get(refs, "graphic-properties", key) }
	if get("draw:shadow") != "visible" {
		return nil
	}
	c := parseColor(get("draw:shadow-color"))
	if c == "" {
		c = "#808080"
	}
	if v, ok := parsePercent(get("draw:shadow-opacity")); ok {
		c = withAlpha(c, v)
	}
	x, _ := s.length(get("draw:shadow-offset-x"))
	y, _ := s.length(get("draw:shadow-offset-y"))
	blur, _ := s.length(get("draw:shadow-blur"))
	return &outShadow{Color: c, X: round(x), Y: round(y), Blur: round(math.Max(blur, 0))}
}
