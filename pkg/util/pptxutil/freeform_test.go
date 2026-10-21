package pptxutil_test

// cspell:words arcTo avLst cubicBezTo custGeom extrusionOk fmla gdLst lnTo moveTo pathLst quadBezTo srgb stAng swAng wR hR xfrm

import (
	"bytes"
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/pptxutil"
)

// freeform is a shape holding custGeom with paths, at a 200 by 150 frame
// (2,540,000 by 1,905,000 EMU on the test deck's 1920-wide slide).
func freeform(xfrmAttrs, paths string) string {
	return `<p:sp><p:nvSpPr><p:cNvPr id="2" name="Freeform"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>` +
		`<p:spPr><a:xfrm` + xfrmAttrs + `><a:off x="1270000" y="635000"/><a:ext cx="2540000" cy="1905000"/></a:xfrm>` +
		`<a:custGeom><a:avLst/><a:gdLst/><a:rect b="b" l="l" r="r" t="t"/><a:pathLst>` + paths + `</a:pathLst></a:custGeom>` +
		`<a:solidFill><a:srgbClr val="3366FF"/></a:solidFill>` +
		`<a:ln w="12700"><a:solidFill><a:srgbClr val="112233"/></a:solidFill></a:ln></p:spPr></p:sp>`
}

// importFreeform imports one freeform and returns its element and the
// import's warnings.
func importFreeform(t *testing.T, xfrmAttrs, paths string) (map[string]any, []string) {
	t.Helper()
	doc, result := importPptx(t, deck(freeform(xfrmAttrs, paths)).zip(t), nil)
	var messages []string
	for _, w := range result.Warnings {
		messages = append(messages, w.Message)
	}
	els := elements(slides(doc)[0])
	// The master's ellipse, then the freeform.
	if len(els) != 2 {
		t.Fatalf("elements = %s", asJSON(els))
	}
	return els[1], messages
}

// sameContours fails unless got, a decoded "contours" value, matches want
// to within a millionth.
func sameContours(t *testing.T, got any, want string) {
	t.Helper()
	var w any
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("want: %v", err)
	}
	if !near(got, w) {
		t.Errorf("contours =\n%s\nwant\n%s", asJSON(got), asJSON(w))
	}
}

func near(a, b any) bool {
	switch b := b.(type) {
	case float64:
		a, ok := a.(float64)
		return ok && math.Abs(a-b) < 1e-6
	case []any:
		a, ok := a.([]any)
		if !ok || len(a) != len(b) {
			return false
		}
		for i := range b {
			if !near(a[i], b[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		a, ok := a.(map[string]any)
		if !ok || len(a) != len(b) {
			return false
		}
		for k := range b {
			if !near(a[k], b[k]) {
				return false
			}
		}
		return true
	}
	return a == b
}

// The five-point outline from #3631, as a real deck stored it.
func TestImportDrawsAFreeformAsItsOutline(t *testing.T) {
	el, warnings := importFreeform(t, "", `<a:path extrusionOk="0" h="48898" w="84254">`+
		`<a:moveTo><a:pt x="0" y="0"/></a:moveTo>`+
		`<a:lnTo><a:pt x="50319" y="0"/></a:lnTo>`+
		`<a:lnTo><a:pt x="84254" y="33935"/></a:lnTo>`+
		`<a:lnTo><a:pt x="84254" y="48898"/></a:lnTo>`+
		`<a:lnTo><a:pt x="48798" y="48898"/></a:lnTo>`+
		`<a:close/></a:path>`)
	if len(warnings) != 0 {
		t.Errorf("warnings = %q", warnings)
	}
	if el["kind"] != "path" || el["preset"] != nil || el["fill"] != "#3366FF" {
		t.Errorf("freeform = %s", asJSON(el))
	}
	sameContours(t, el["contours"], `[{"commands": [
		{"op": "move", "x": 0, "y": 0},
		{"op": "line", "x": 0.59723, "y": 0},
		{"op": "line", "x": 1, "y": 0.693996},
		{"op": "line", "x": 1, "y": 1},
		{"op": "line", "x": 0.579177, "y": 1},
		{"op": "close"}]}]`)
}

func TestImportReadsFreeformCurvesArcsAndFlags(t *testing.T) {
	// A 200 by 150 frame over a 100 by 100 path: x scales by 2, y by 1.5.
	el, warnings := importFreeform(t, "", `<a:path w="100" h="100" fill="none">`+
		`<a:moveTo><a:pt x="0" y="50"/></a:moveTo>`+
		`<a:quadBezTo><a:pt x="0" y="0"/><a:pt x="50" y="0"/></a:quadBezTo>`+
		`<a:cubicBezTo><a:pt x="100" y="0"/><a:pt x="100" y="100"/><a:pt x="50" y="100"/></a:cubicBezTo>`+
		`<a:arcTo wR="50" hR="50" stAng="5400000" swAng="5400000"/>`+
		`</a:path>`+
		`<a:path w="100" h="100" stroke="0">`+
		`<a:moveTo><a:pt x="25" y="25"/></a:moveTo>`+
		`<a:lnTo><a:pt x="75" y="25"/></a:lnTo>`+
		`<a:arcTo wR="10" hR="10" stAng="2700000" swAng="-5400000"/>`+
		`<a:close/></a:path>`)
	if len(warnings) != 0 {
		t.Errorf("warnings = %q", warnings)
	}
	// 90 degrees stays 90 in any aspect. 45 degrees on a circle stretched
	// 2 by 1.5 points at atan2(1.5, 2) = 36.869898; -45 at -36.869898.
	sameContours(t, el["contours"], `[
		{"commands": [
			{"op": "move", "x": 0, "y": 0.5},
			{"op": "quad", "x1": 0, "y1": 0, "x": 0.5, "y": 0},
			{"op": "cubic", "x1": 1, "y1": 0, "x2": 1, "y2": 1, "x": 0.5, "y": 1},
			{"op": "arc", "wR": 0.5, "hR": 0.5, "start": 90, "swing": 90}],
		 "fill": false},
		{"commands": [
			{"op": "move", "x": 0.25, "y": 0.25},
			{"op": "line", "x": 0.75, "y": 0.25},
			{"op": "arc", "wR": 0.1, "hR": 0.1, "start": 36.869898, "swing": -73.739795},
			{"op": "close"}],
		 "stroke": false}]`)
}

func TestImportMirrorsAFlippedFreeform(t *testing.T) {
	el, _ := importFreeform(t, ` flipH="1"`, `<a:path w="100" h="100">`+
		`<a:moveTo><a:pt x="0" y="0"/></a:moveTo>`+
		`<a:lnTo><a:pt x="100" y="0"/></a:lnTo>`+
		`<a:arcTo wR="50" hR="50" stAng="0" swAng="5400000"/>`+
		`<a:close/></a:path>`)
	sameContours(t, el["contours"], `[{"commands": [
		{"op": "move", "x": 1, "y": 0},
		{"op": "line", "x": 0, "y": 0},
		{"op": "arc", "wR": 0.5, "hR": 0.5, "start": 180, "swing": -90},
		{"op": "close"}]}]`)

	el, _ = importFreeform(t, ` flipV="1"`, `<a:path w="100" h="100">`+
		`<a:moveTo><a:pt x="0" y="0"/></a:moveTo>`+
		`<a:arcTo wR="50" hR="50" stAng="0" swAng="5400000"/></a:path>`)
	sameContours(t, el["contours"], `[{"commands": [
		{"op": "move", "x": 0, "y": 1},
		{"op": "arc", "wR": 0.5, "hR": 0.5, "start": 0, "swing": -90}]}]`)
}

func TestImportScalesAFreeformWithoutAPathSizeToTheShape(t *testing.T) {
	el, _ := importFreeform(t, "", `<a:path>`+
		`<a:moveTo><a:pt x="0" y="0"/></a:moveTo>`+
		`<a:lnTo><a:pt x="1270000" y="1905000"/></a:lnTo></a:path>`)
	sameContours(t, el["contours"], `[{"commands": [
		{"op": "move", "x": 0, "y": 0},
		{"op": "line", "x": 0.5, "y": 1}]}]`)
}

func TestImportResolvesLiteralFreeformGuides(t *testing.T) {
	doc, result := importPptx(t, deck(strings.Replace(freeform("", `<a:path w="100" h="100">`+
		`<a:moveTo><a:pt x="0" y="0"/></a:moveTo>`+
		`<a:lnTo><a:pt x="half" y="100"/></a:lnTo></a:path>`),
		`<a:gdLst/>`, `<a:gdLst><a:gd name="half" fmla="val 50"/></a:gdLst>`, 1)).zip(t), nil)
	if len(result.Warnings) != 0 {
		t.Errorf("warnings = %v", result.Warnings)
	}
	sameContours(t, elements(slides(doc)[0])[1]["contours"], `[{"commands": [
		{"op": "move", "x": 0, "y": 0},
		{"op": "line", "x": 0.5, "y": 1}]}]`)
}

func TestImportDrawsAnUnreadableFreeformAsARectangle(t *testing.T) {
	for name, paths := range map[string]string{
		"no paths": ``,
		"a formula guide": `<a:path w="100" h="100"><a:moveTo><a:pt x="0" y="0"/></a:moveTo>` +
			`<a:lnTo><a:pt x="unknown" y="100"/></a:lnTo></a:path>`,
		"a curve short a point": `<a:path w="100" h="100"><a:moveTo><a:pt x="0" y="0"/></a:moveTo>` +
			`<a:cubicBezTo><a:pt x="1" y="1"/></a:cubicBezTo></a:path>`,
	} {
		el, warnings := importFreeform(t, "", paths)
		if el["kind"] != "rectangle" || el["contours"] != nil {
			t.Errorf("%s: %s", name, asJSON(el))
		}
		if len(warnings) != 1 || warnings[0] != "Freeform shapes that could not be read were drawn as rectangles." {
			t.Errorf("%s: warnings = %q", name, warnings)
		}
	}
}

// A contour-only path shape exports as custGeom in the shape's own EMU and
// imports back as the same contours.
func TestFreeformRoundTrips(t *testing.T) {
	const contours = `[
		{"commands": [
			{"op": "move", "x": 0, "y": 0.5},
			{"op": "line", "x": 0.25, "y": 0},
			{"op": "quad", "x1": 0.5, "y1": 0, "x": 0.5, "y": 0.25},
			{"op": "cubic", "x1": 0.75, "y1": 0, "x2": 1, "y2": 0.25, "x": 1, "y": 0.5},
			{"op": "arc", "wR": 0.5, "hR": 0.5, "start": 0, "swing": 90},
			{"op": "close"}]},
		{"commands": [
			{"op": "move", "x": 0.25, "y": 0.75},
			{"op": "line", "x": 0.75, "y": 0.75}],
		 "fill": false},
		{"commands": [
			{"op": "move", "x": 0, "y": 0},
			{"op": "line", "x": 0.5, "y": 0.5}],
		 "stroke": false}]`
	source := `{
	  "schemaVersion": 5,
	  "size": {"width": 1920, "height": 1080},
	  "slides": [{"id": "s1", "elements": [
	    {"id": "e1", "type": "shape", "kind": "path",
	     "frame": {"x": 200, "y": 100, "width": 400, "height": 200},
	     "fill": "#3366FF", "stroke": {"color": "#112233", "width": 2},
	     "contours": ` + contours + `}
	  ]}]
	}`
	var exported bytes.Buffer
	if _, err := pptxutil.ExportQslide(pptxutil.ExportQslideParams{
		Source: strings.NewReader(source), Out: &exported,
	}); err != nil {
		t.Fatalf("ExportQslide: %v", err)
	}
	slide := unzip(t, exported.Bytes()).part(t, "ppt/slides/slide1.xml")
	contains(t, "slide", slide,
		`<a:custGeom><a:avLst/><a:gdLst/><a:ahLst/><a:cxnLst/><a:rect l="l" t="t" r="r" b="b"/><a:pathLst>`,
		`<a:path w="2540000" h="1270000"><a:moveTo><a:pt x="0" y="635000"/></a:moveTo>`,
		`<a:arcTo wR="1270000" hR="635000" stAng="0" swAng="5400000"/><a:close/></a:path>`,
		`<a:path w="2540000" h="1270000" fill="none">`,
		`<a:path w="2540000" h="1270000" stroke="0">`,
	)
	doc, result := importPptx(t, exported.Bytes(), nil)
	if len(result.Warnings) != 0 {
		t.Errorf("warnings = %v", result.Warnings)
	}
	var found bool
	for _, el := range elements(slides(doc)[0]) {
		if el["kind"] == "path" {
			found = true
			sameContours(t, el["contours"], contours)
		}
	}
	if !found {
		t.Errorf("no path shape: %s", asJSON(elements(slides(doc)[0])))
	}
}
