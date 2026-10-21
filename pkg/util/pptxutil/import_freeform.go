package pptxutil

// cspell:ignore Bez cust fmla Geom xfrm

import (
	"math"
	"strconv"
	"strings"
)

// freeformContours converts a freeform's paths into contours in its frame's
// own space, mirrored as the transform flips the shape. Each path's w by h
// coordinates are scaled to the frame. ok is false when a path uses a
// command or a guide this cannot read, or nothing is drawn at all.
func freeformContours(geom *xCustomGeometry, xfrm *xXfrm, frame outFrame) (contours []shapeContour, ok bool) {
	guides := map[string]float64{}
	for _, gd := range append(append([]xGuide{}, geom.Adjust...), geom.Guides...) {
		v, isVal := strings.CutPrefix(strings.TrimSpace(gd.Formula), "val ")
		if n, err := strconv.ParseFloat(strings.TrimSpace(v), 64); isVal && err == nil && gd.Name != "" {
			guides[gd.Name] = n
		}
	}
	for _, p := range geom.Paths {
		contour, ok := freeformPath(p, guides, xfrm, frame)
		if !ok {
			return nil, false
		}
		if len(contour.Commands) > 0 {
			contours = append(contours, contour)
		}
	}
	return contours, len(contours) > 0
}

// freeformPath converts one path; see freeformContours.
func freeformPath(p xPath, guides map[string]float64, xfrm *xXfrm, frame outFrame) (shapeContour, bool) {
	w, h := float64(p.W), float64(p.H)
	if w <= 0 {
		w = float64(xfrm.Ext.CX)
	}
	if h <= 0 {
		h = float64(xfrm.Ext.CY)
	}
	flipH, flipV := bool(xfrm.FlipH), bool(xfrm.FlipV)
	fraction := func(v, size float64, flip bool) float64 {
		f := 0.0
		if size > 0 {
			f = v / size
		}
		if flip {
			f = 1 - f
		}
		return precise(f)
	}
	// How far the frame stretches the path one way against the other, and
	// which way it mirrors it: an arc's angles follow.
	sx, sy := 1.0, 1.0
	if w > 0 && h > 0 && frame.Width > 0 && frame.Height > 0 {
		sx, sy = frame.Width/w, frame.Height/h
	}
	if flipH {
		sx = -sx
	}
	if flipV {
		sy = -sy
	}

	value := func(s string) (float64, bool) {
		if n, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
			return n, true
		}
		n, ok := guides[s]
		return n, ok
	}
	contour := shapeContour{}
	if p.Fill == "none" {
		contour.Fill = new(bool)
	}
	if p.Stroke != nil && !bool(*p.Stroke) {
		contour.Stroke = new(bool)
	}
	for _, c := range p.Commands {
		var pts []float64
		for _, pt := range c.Points {
			x, okX := value(pt.X)
			y, okY := value(pt.Y)
			if !okX || !okY {
				return shapeContour{}, false
			}
			pts = append(pts, fraction(x, w, flipH), fraction(y, h, flipV))
		}
		var cmd pathCommand
		switch {
		case c.XMLName.Local == "moveTo" && len(pts) == 2:
			cmd = pathCommand{Op: "move", X: pts[0], Y: pts[1]}
		case c.XMLName.Local == "lnTo" && len(pts) == 2:
			cmd = pathCommand{Op: "line", X: pts[0], Y: pts[1]}
		case c.XMLName.Local == "quadBezTo" && len(pts) == 4:
			cmd = pathCommand{Op: "quad", X1: pts[0], Y1: pts[1], X: pts[2], Y: pts[3]}
		case c.XMLName.Local == "cubicBezTo" && len(pts) == 6:
			cmd = pathCommand{Op: "cubic", X1: pts[0], Y1: pts[1], X2: pts[2], Y2: pts[3], X: pts[4], Y: pts[5]}
		case c.XMLName.Local == "arcTo":
			wR, okW := value(c.WR)
			hR, okH := value(c.HR)
			st, okS := value(c.StAng)
			sw, okA := value(c.SwAng)
			if !okW || !okH || !okS || !okA {
				return shapeContour{}, false
			}
			start, swing := arcAngles(st/60_000, sw/60_000, sx, sy)
			cmd = pathCommand{
				Op: "arc", WR: fraction(wR, w, false), HR: fraction(hR, h, false),
				Start: precise(start), Swing: precise(swing),
			}
		case c.XMLName.Local == "close":
			cmd = pathCommand{Op: "close"}
		default:
			return shapeContour{}, false
		}
		contour.Commands = append(contour.Commands, cmd)
	}
	return contour, true
}

// arcAngles carries an arc's start and swing, in degrees clockwise from 3
// o'clock, from a path's space into its frame's, which stretches the path
// by sx and sy and mirrors it where either is negative.
//
// An arc's angles name rays from the ellipse's center, and stretching
// turns a ray: 45 degrees on a circle stretched twice as wide points at
// about 27. The start is within one turn; the swing keeps its full turns
// and runs the other way under a mirror.
func arcAngles(start, swing, sx, sy float64) (float64, float64) {
	ray := func(deg float64) float64 {
		r := deg * math.Pi / 180
		return math.Atan2(sy*math.Sin(r), sx*math.Cos(r)) * 180 / math.Pi
	}
	mirrored := (sx < 0) != (sy < 0)
	rest := math.Mod(swing, 360)
	turns := swing - rest
	if mirrored {
		rest, turns = -rest, -turns
	}
	from := ray(start)
	sweep := 0.0
	if rest != 0 {
		sweep = ray(start+swing) - from
		if rest > 0 && sweep <= 0 {
			sweep += 360
		}
		if rest < 0 && sweep >= 0 {
			sweep -= 360
		}
	}
	from = math.Mod(from, 360)
	if from < 0 {
		from += 360
	}
	return from, sweep + turns
}

// precise rounds a frame fraction or an angle to a millionth, never -0.
func precise(v float64) float64 {
	r := math.Round(v*1e6) / 1e6
	if r == 0 {
		return 0
	}
	return r
}
