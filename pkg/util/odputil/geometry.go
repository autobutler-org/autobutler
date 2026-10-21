package odputil

import (
	"encoding/xml"
	"math"
	"strconv"
	"strings"
)

// maxTransformOps bounds the operations one draw:transform applies.
const maxTransformOps = 16

// affine is a 2D transform, x' = a·x + c·y + e and y' = b·x + d·y + f, in
// cm.
type affine struct{ a, b, c, d, e, f float64 }

// identityAffine leaves every point where it is.
var identityAffine = affine{a: 1, d: 1}

// then is m followed by n.
func (m affine) then(n affine) affine {
	return affine{
		a: n.a*m.a + n.c*m.b, b: n.b*m.a + n.d*m.b,
		c: n.a*m.c + n.c*m.d, d: n.b*m.c + n.d*m.d,
		e: n.a*m.e + n.c*m.f + n.e, f: n.b*m.e + n.d*m.f + n.f,
	}
}

// apply maps a point.
func (m affine) apply(x, y float64) (float64, float64) {
	return m.a*x + m.c*y + m.e, m.b*x + m.d*y + m.f
}

// parseTransform reads a draw:transform, its operations applied in the
// order they are written: rotate (radians, counterclockwise as drawn),
// translate and matrix (lengths), scale and skewX and skewY. An operation it
// cannot read ends the list there.
func parseTransform(v string) affine {
	m := identityAffine
	for range maxTransformOps {
		v = strings.TrimSpace(v)
		open := strings.IndexByte(v, '(')
		end := strings.IndexByte(v, ')')
		if open <= 0 || end < open {
			break
		}
		name := strings.TrimSpace(v[:open])
		args := strings.FieldsFunc(v[open+1:end], func(r rune) bool { return r == ' ' || r == ',' })
		v = v[end+1:]
		op, ok := transformOp(name, args)
		if !ok {
			break
		}
		m = m.then(op)
	}
	return m
}

// transformOp is one draw:transform operation.
func transformOp(name string, args []string) (affine, bool) {
	num := func(i int) (float64, bool) {
		if i >= len(args) {
			return 0, false
		}
		n, err := strconv.ParseFloat(args[i], 64)
		return n, err == nil && !math.IsNaN(n) && !math.IsInf(n, 0) && math.Abs(n) < 1e6
	}
	length := func(i int) (float64, bool) {
		if i >= len(args) {
			return 0, false
		}
		return parseCm(args[i])
	}
	switch name {
	case "rotate":
		t, ok := num(0)
		sin, cos := math.Sincos(t)
		return affine{a: cos, b: -sin, c: sin, d: cos}, ok
	case "translate":
		x, ok := length(0)
		y, okY := length(1)
		if !okY {
			y = 0
		}
		return affine{a: 1, d: 1, e: x, f: y}, ok
	case "scale":
		x, ok := num(0)
		y, okY := num(1)
		if !okY {
			y = x
		}
		return affine{a: x, d: y}, ok
	case "skewX":
		t, ok := num(0)
		return affine{a: 1, c: math.Tan(t), d: 1}, ok
	case "skewY":
		t, ok := num(0)
		return affine{a: 1, b: math.Tan(t), d: 1}, ok
	case "matrix":
		var vals [4]float64
		for i := range vals {
			n, ok := num(i)
			if !ok {
				return affine{}, false
			}
			vals[i] = n
		}
		e, okE := length(4)
		f, okF := length(5)
		return affine{a: vals[0], b: vals[1], c: vals[2], d: vals[3], e: e, f: f}, okE && okF
	}
	return affine{}, false
}

// box reads an element's svg:x, svg:y, svg:width, svg:height and
// draw:transform into a frame in slide units: the transform moves the
// box's center and turns it, and its scale stretches the box.
func (s *slideReader) box(c xml.StartElement) outFrame {
	x, _ := parseCm(attr(c, nsSVG, "x"))
	y, _ := parseCm(attr(c, nsSVG, "y"))
	w, _ := parseCm(attr(c, nsSVG, "width"))
	h, _ := parseCm(attr(c, nsSVG, "height"))
	w, h = math.Max(w, 0), math.Max(h, 0)
	var rotation float64
	if t := attr(c, nsDraw, "transform"); t != "" {
		m := parseTransform(t)
		cx, cy := m.apply(x+w/2, y+h/2)
		w *= math.Hypot(m.a, m.b)
		h *= math.Hypot(m.c, m.d)
		x, y = cx-w/2, cy-h/2
		rotation = math.Mod(math.Atan2(m.b, m.a)*180/math.Pi, 360)
		if rotation < 0 {
			rotation += 360
		}
	}
	frame := outFrame{X: s.units(x), Y: s.units(y), Width: s.units(w), Height: s.units(h), Rotation: round(rotation)}
	if frame.Rotation >= 360 {
		frame.Rotation = 0
	}
	return frame
}

// endpoints reads a line's svg:x1, svg:y1, svg:x2 and svg:y2, through its
// draw:transform, in cm.
func endpoints(c xml.StartElement) (x1, y1, x2, y2 float64) {
	x1, _ = parseCm(attr(c, nsSVG, "x1"))
	y1, _ = parseCm(attr(c, nsSVG, "y1"))
	x2, _ = parseCm(attr(c, nsSVG, "x2"))
	y2, _ = parseCm(attr(c, nsSVG, "y2"))
	if t := attr(c, nsDraw, "transform"); t != "" {
		m := parseTransform(t)
		x1, y1 = m.apply(x1, y1)
		x2, y2 = m.apply(x2, y2)
	}
	return x1, y1, x2, y2
}

// inset shrinks a frame by its padding, as the editor draws text flush
// with its frame.
func (s *slideReader) inset(frame outFrame, refs []styleRef) outFrame {
	side := func(key string) float64 {
		v, ok := s.styles.prop(s.scope, refs, "fo:padding-"+key)
		if !ok {
			v, _ = s.styles.prop(s.scope, refs, "fo:padding")
		}
		cm, _ := parseCm(v)
		return s.units(math.Max(cm, 0))
	}
	l, r, t, b := side("left"), side("right"), side("top"), side("bottom")
	if l+r < frame.Width {
		frame.X, frame.Width = round(frame.X+l), round(frame.Width-l-r)
	}
	if t+b < frame.Height {
		frame.Y, frame.Height = round(frame.Y+t), round(frame.Height-t-b)
	}
	return frame
}
