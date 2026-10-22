package pptxutil_test

// cspell:words AABBCC avLst diagStripe flowChartDelay fmla prst round2SameRect rtTriangle srgb xfrm

import (
	"strings"
	"testing"
)

// The presets a real deck drew as rectangles, as the file stored them.
func TestImportDrawsKnownPresetsAsPathShapes(t *testing.T) {
	geom := func(prst, guides string) string {
		return `<p:sp><p:nvSpPr><p:cNvPr id="2" name="S"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>` +
			`<p:spPr><a:xfrm><a:off x="1270000" y="635000"/><a:ext cx="2540000" cy="1905000"/></a:xfrm>` +
			`<a:prstGeom prst="` + prst + `"><a:avLst>` + guides + `</a:avLst></a:prstGeom>` +
			`<a:solidFill><a:srgbClr val="3366FF"/></a:solidFill>` +
			`<a:ln w="12700"><a:solidFill><a:srgbClr val="112233"/></a:solidFill></a:ln></p:spPr></p:sp>`
	}
	cases := []struct {
		prst   string
		guides string
		adjust map[string]float64
	}{
		{"diagStripe", `<a:gd name="adj" fmla="val 50000"/>`, map[string]float64{"adj": 50000}},
		{"trapezoid", `<a:gd name="adj" fmla="val 25000"/>`, map[string]float64{"adj": 25000}},
		{"rtTriangle", ``, nil},
		{"round2SameRect", `<a:gd name="adj1" fmla="val 4129"/><a:gd name="adj2" fmla="val 0"/>`,
			map[string]float64{"adj1": 4129, "adj2": 0}},
		{"flowChartDelay", ``, nil},
	}
	var body strings.Builder
	for _, tc := range cases {
		body.WriteString(geom(tc.prst, tc.guides))
	}
	// Still no outline for a preset the editor cannot evaluate.
	body.WriteString(geom("hexagon", `<a:gd name="adj" fmla="val 25000"/>`))

	doc, result := importPptx(t, deck(body.String()).zip(t), nil)
	var messages []string
	for _, w := range result.Warnings {
		messages = append(messages, w.Message)
	}
	if len(messages) != 1 || messages[0] != "Shapes of a kind the editor does not have (hexagon) were drawn as rectangles." {
		t.Errorf("warnings = %q", messages)
	}
	els := elements(slides(doc)[0])
	// The master's ellipse, then the shapes in order.
	if len(els) != len(cases)+2 {
		t.Fatalf("elements = %s", asJSON(els))
	}
	for i, tc := range cases {
		el := els[i+1]
		if el["kind"] != "path" || el["preset"] != tc.prst {
			t.Errorf("%s = %s", tc.prst, asJSON(el))
		}
		if el["fill"] != "#3366FF" {
			t.Errorf("%s fill = %v", tc.prst, el["fill"])
		}
		got, _ := el["adjust"].(map[string]any)
		if len(tc.adjust) == 0 {
			if len(got) != 0 {
				t.Errorf("%s adjust = %v, want none", tc.prst, got)
			}
			continue
		}
		if len(got) != len(tc.adjust) {
			t.Errorf("%s adjust = %v", tc.prst, got)
			continue
		}
		for name, want := range tc.adjust {
			if got[name] != want {
				t.Errorf("%s adjust %s = %v, want %v", tc.prst, name, got[name], want)
			}
		}
	}
	if els[len(cases)+1]["kind"] != "rectangle" || els[len(cases)+1]["preset"] != nil {
		t.Errorf("hexagon = %s", asJSON(els[len(cases)+1]))
	}
}

func TestImportDrawsALayoutPresetAsAPath(t *testing.T) {
	b := deck(``)
	b["ppt/slideLayouts/slideLayout1.xml"] = `<p:sldLayout ` + ns + `><p:cSld><p:spTree>` +
		`<p:sp><p:nvSpPr><p:cNvPr id="2" name="Title"/><p:cNvSpPr/><p:nvPr><p:ph type="ctrTitle"/></p:nvPr></p:nvSpPr>` +
		`<p:spPr/></p:sp>` +
		`<p:sp><p:nvSpPr><p:cNvPr id="3" name="Stripe"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>` +
		`<p:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="12192000" cy="685800"/></a:xfrm>` +
		`<a:prstGeom prst="diagStripe"><a:avLst><a:gd name="adj" fmla="val 50000"/></a:avLst></a:prstGeom>` +
		`<a:solidFill><a:srgbClr val="AABBCC"/></a:solidFill></p:spPr></p:sp>` +
		`</p:spTree></p:cSld></p:sldLayout>`
	doc, result := importPptx(t, b.zip(t), nil)
	if len(result.Warnings) != 0 {
		t.Errorf("warnings = %v", result.Warnings)
	}
	var found bool
	for _, el := range elements(slides(doc)[0]) {
		if el["preset"] == "diagStripe" && el["kind"] == "path" && el["fill"] == "#AABBCC" {
			found = true
		}
	}
	if !found {
		t.Errorf("layout stripe missing: %s", asJSON(elements(slides(doc)[0])))
	}
}

func TestExportWritesAPathPresetBack(t *testing.T) {
	deck := `{
	  "schemaVersion": 1,
	  "size": {"width": 1920, "height": 1080},
	  "slides": [{"id": "s1", "elements": [
	    {"id": "e1", "type": "shape", "kind": "path", "preset": "round2SameRect",
	     "adjust": {"adj2": 0, "adj1": 4129},
	     "frame": {"x": 200, "y": 100, "width": 400, "height": 300},
	     "fill": "#3366FF", "stroke": {"color": "#112233", "width": 2}},
	    {"id": "e2", "type": "shape", "kind": "path", "preset": "rtTriangle",
	     "frame": {"x": 0, "y": 0, "width": 100, "height": 100}, "fill": "#00FF00"}
	  ]}]
	}`
	parts, _ := export(t, deck, nil)
	slide := parts.part(t, "ppt/slides/slide1.xml")
	contains(t, "slide", slide,
		`<a:prstGeom prst="round2SameRect"><a:avLst><a:gd name="adj1" fmla="val 4129"/><a:gd name="adj2" fmla="val 0"/></a:avLst></a:prstGeom>`,
		`<a:prstGeom prst="rtTriangle"><a:avLst/></a:prstGeom>`,
		`<a:srgbClr val="3366FF"/>`,
		`<a:ln w="12700"><a:solidFill><a:srgbClr val="112233"/></a:solidFill></a:ln>`,
	)
}
