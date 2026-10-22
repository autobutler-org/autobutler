package pptxutil_test

// cspell:words algn Lato Montserrat prst Shdw srgb xfrm cxn

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/pptxutil"
)

// shadowedShape is a rectangle with the outer shadow #3632 found in a real
// deck, or with effects when they are given in its place.
func shadowedShape(id string, effects ...string) string {
	effect := `<a:effectLst><a:outerShdw blurRad="387350" dist="38100" dir="8100000" algn="tr" sx="107000" sy="107000" rotWithShape="0">` +
		`<a:srgbClr val="000000"><a:alpha val="49800"/></a:srgbClr></a:outerShdw></a:effectLst>`
	if len(effects) > 0 {
		effect = strings.Join(effects, "")
	}
	return `<p:sp><p:nvSpPr><p:cNvPr id="` + id + `" name="S"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>` +
		`<p:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="635000" cy="635000"/></a:xfrm>` +
		`<a:prstGeom prst="rect"><a:avLst/></a:prstGeom><a:solidFill><a:srgbClr val="FF0000"/></a:solidFill>` +
		effect + `</p:spPr></p:sp>`
}

// embedFonts adds an embeddedFontLst naming typefaces to the deck's
// presentation part.
func embedFonts(b pptxBuilder, typefaces ...string) {
	var list strings.Builder
	list.WriteString(`<p:embeddedFontLst>`)
	for i, typeface := range typefaces {
		list.WriteString(`<p:embeddedFont><p:font typeface="` + typeface + `"/>`)
		if i == 0 {
			list.WriteString(`<p:regular r:id="rId29"/><p:bold r:id="rId30"/><p:italic r:id="rId31"/><p:boldItalic r:id="rId32"/>`)
		}
		list.WriteString(`</p:embeddedFont>`)
	}
	list.WriteString(`</p:embeddedFontLst>`)
	b["ppt/presentation.xml"] = strings.Replace(b["ppt/presentation.xml"], `</p:presentation>`,
		list.String()+`</p:presentation>`, 1)
}

// exactShadow is an outer shadow with no scale or skew: 10 units at 135
// degrees, a 20 unit blur, in half-transparent black.
const exactShadow = `<a:effectLst><a:outerShdw blurRad="127000" dist="63500" dir="8100000" algn="tr">` +
	`<a:srgbClr val="000000"><a:alpha val="50000"/></a:srgbClr></a:outerShdw></a:effectLst>`

// shadowOf returns the element's decoded shadow, failing when it has none.
func shadowOf(t *testing.T, el map[string]any) map[string]any {
	t.Helper()
	shadow, ok := el["shadow"].(map[string]any)
	if !ok {
		t.Fatalf("no shadow: %s", asJSON(el))
	}
	return shadow
}

func TestImportDrawsAnOuterShadow(t *testing.T) {
	doc, result := importPptx(t, deck(shadowedShape("2", exactShadow)).zip(t), nil)
	if len(result.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", result.Warnings)
	}
	els := elements(slides(doc)[0])
	want := map[string]any{"color": "#00000080", "x": -7.07, "y": 7.07, "blur": float64(20)}
	if got := shadowOf(t, els[len(els)-1]); !near(got, want) {
		t.Errorf("shadow = %s, want %s", asJSON(got), asJSON(want))
	}
}

func TestImportReadsTheShadowFromIssue3632(t *testing.T) {
	doc, _ := importPptx(t, deck(shadowedShape("2")).zip(t), nil)
	els := elements(slides(doc)[0])
	// blurRad 387350 and dist 38100 EMU at 6350 EMU a unit, at 135 degrees;
	// alpha 49.8% is 0x7F.
	want := map[string]any{"color": "#0000007F", "x": -4.24, "y": 4.24, "blur": float64(61)}
	if got := shadowOf(t, els[len(els)-1]); !near(got, want) {
		t.Errorf("shadow = %s, want %s", asJSON(got), asJSON(want))
	}
}

func TestImportTurnsAShadowWithItsShape(t *testing.T) {
	for _, tc := range []struct {
		name, attrs string
		x, y        float64
	}{
		// rotWithShape defaults to true: a quarter turn sends the shadow down.
		{"turning", ``, 0, 10},
		{"fixed", ` rotWithShape="0"`, 10, 0},
	} {
		shape := `<p:sp><p:nvSpPr><p:cNvPr id="2" name="S"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>` +
			`<p:spPr><a:xfrm rot="5400000"><a:off x="0" y="0"/><a:ext cx="635000" cy="635000"/></a:xfrm>` +
			`<a:prstGeom prst="rect"><a:avLst/></a:prstGeom><a:solidFill><a:srgbClr val="FF0000"/></a:solidFill>` +
			`<a:effectLst><a:outerShdw dist="63500" dir="0"` + tc.attrs + `><a:prstClr val="black"/></a:outerShdw></a:effectLst>` +
			`</p:spPr></p:sp>`
		doc, _ := importPptx(t, deck(shape).zip(t), nil)
		els := elements(slides(doc)[0])
		want := map[string]any{"color": "#000000", "x": tc.x, "y": tc.y}
		got := shadowOf(t, els[len(els)-1])
		for _, k := range []string{"x", "y"} {
			if _, ok := got[k]; !ok {
				got[k] = float64(0)
			}
		}
		if !near(got, want) {
			t.Errorf("%s: shadow = %s, want %s", tc.name, asJSON(got), asJSON(want))
		}
	}
}

func TestImportDrawsAPictureShadow(t *testing.T) {
	picture := `<p:pic><p:nvPicPr><p:cNvPr id="9" name="P"/><p:cNvPicPr/><p:nvPr/></p:nvPicPr>` +
		`<p:blipFill><a:blip r:embed="rId2"/></p:blipFill><p:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="6350" cy="6350"/></a:xfrm>` +
		exactShadow + `</p:spPr></p:pic>`
	b := deck(picture)
	b["ppt/media/image1.png"] = string(pngBytes(t, 1, 1))
	doc, result := importPptx(t, b.zip(t), &mediaStore{})
	if len(result.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", result.Warnings)
	}
	els := elements(slides(doc)[0])
	if got := shadowOf(t, els[len(els)-1]); got["color"] != "#00000080" {
		t.Errorf("shadow = %s", asJSON(got))
	}
}

func TestImportStillWarnsOfALineShadow(t *testing.T) {
	line := `<p:cxnSp><p:nvCxnSpPr><p:cNvPr id="4" name="L"/><p:cNvCxnSpPr/><p:nvPr/></p:nvCxnSpPr>` +
		`<p:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="635000" cy="0"/></a:xfrm>` +
		`<a:prstGeom prst="line"><a:avLst/></a:prstGeom><a:ln w="12700"><a:solidFill><a:srgbClr val="000000"/></a:solidFill></a:ln>` +
		exactShadow + `</p:spPr></p:cxnSp>`
	doc, result := importPptx(t, deck(line).zip(t), nil)
	want := []pptxutil.ImportWarning{{Slide: 1, Message: "The shadow on 1 shape was left out."}}
	if got := result.Warnings; !reflect.DeepEqual(got, want) {
		t.Errorf("warnings = %v, want %v", got, want)
	}
	for _, el := range elements(slides(doc)[0]) {
		if _, ok := el["shadow"]; ok {
			t.Errorf("a line has a shadow: %s", asJSON(el))
		}
	}
}

// A shadow exports as an outer shadow fixed to the slide, the shape's
// opacity folded into its alpha, and imports back as the same shadow.
func TestShadowRoundTrips(t *testing.T) {
	source := `{
	  "schemaVersion": 6,
	  "size": {"width": 1920, "height": 1080},
	  "slides": [{"id": "s1", "elements": [
	    {"id": "e1", "type": "shape", "kind": "rectangle",
	     "frame": {"x": 200, "y": 100, "width": 400, "height": 200, "rotation": 30},
	     "fill": "#3366FF", "shadow": {"color": "#0000007F", "x": 3, "y": 4, "blur": 10}},
	    {"id": "e2", "type": "shape", "kind": "ellipse",
	     "frame": {"x": 0, "y": 0, "width": 100, "height": 100},
	     "fill": "#3366FF", "opacity": 0.5, "shadow": {"color": "#FF0000", "x": -6}}
	  ]}]
	}`
	parts, _ := export(t, source, nil)
	slide := parts.part(t, "ppt/slides/slide1.xml")
	contains(t, "slide", slide,
		`</a:ln><a:effectLst><a:outerShdw blurRad="63500" dist="31750" dir="3187806" rotWithShape="0">`+
			`<a:srgbClr val="000000"><a:alpha val="49804"/></a:srgbClr></a:outerShdw></a:effectLst></p:spPr>`,
		`<a:effectLst><a:outerShdw blurRad="0" dist="38100" dir="10800000" rotWithShape="0">`+
			`<a:srgbClr val="FF0000"><a:alpha val="50000"/></a:srgbClr></a:outerShdw></a:effectLst>`,
	)
	var exported bytes.Buffer
	if _, err := pptxutil.ExportQslide(pptxutil.ExportQslideParams{
		Source: strings.NewReader(source), Out: &exported,
	}); err != nil {
		t.Fatalf("ExportQslide: %v", err)
	}
	doc, result := importPptx(t, exported.Bytes(), nil)
	if len(result.Warnings) != 0 {
		t.Errorf("warnings = %v", result.Warnings)
	}
	var shadows []any
	for _, el := range elements(slides(doc)[0]) {
		if shadow, ok := el["shadow"]; ok {
			shadows = append(shadows, shadow)
		}
	}
	want := []any{
		map[string]any{"color": "#0000007F", "x": float64(3), "y": float64(4), "blur": float64(10)},
		map[string]any{"color": "#FF000080", "x": float64(-6)},
	}
	if !near(shadows, want) {
		t.Errorf("shadows = %s, want %s", asJSON(shadows), asJSON(want))
	}
}

func TestImportWarnsOnceForEveryShapeShadowLeftOut(t *testing.T) {
	picture := `<p:pic><p:nvPicPr><p:cNvPr id="9" name="P"/><p:cNvPicPr/><p:nvPr/></p:nvPicPr>` +
		`<p:blipFill><a:blip r:embed="rId2"/></p:blipFill><p:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="6350" cy="6350"/></a:xfrm>` +
		`<a:effectLst><a:innerShdw blurRad="63500" dist="50800" dir="2700000"><a:prstClr val="black"/></a:innerShdw></a:effectLst></p:spPr></p:pic>`
	b := deck(
		shadowedShape("2"),
		// No effects, and an empty effect list, are not shadows.
		shadowedShape("2", `<a:effectLst/>`),
		shadowedShape("2")+shadowedShape("3")+picture+
			shadowedShape("5", `<a:effectLst><a:prstShdw prst="shdw1" dist="38100"/></a:effectLst>`),
	)
	b["ppt/media/image1.png"] = string(pngBytes(t, 1, 1))

	_, result := importPptx(t, b.zip(t), &mediaStore{})
	// The inner and preset shadows are left out; the outer ones are drawn
	// without their scale.
	want := []pptxutil.ImportWarning{
		{Slide: 3, Message: "Shadows on 2 shapes were left out."},
		{Slide: 1, Message: "Shadows on 3 shapes may look slightly different."},
		{Slide: 3, Message: "Shadows on 3 shapes may look slightly different."},
	}
	if got := result.Warnings; !reflect.DeepEqual(got, want) {
		t.Errorf("warnings = %v, want %v", got, want)
	}
}

func TestImportCountsAShadowedLayoutShapeOnce(t *testing.T) {
	b := deck(``, ``)
	b["ppt/slideMasters/slideMaster1.xml"] = strings.Replace(b["ppt/slideMasters/slideMaster1.xml"],
		`</p:spTree>`, shadowedShape("8")+`</p:spTree>`, 1)

	_, result := importPptx(t, b.zip(t), nil)
	want := []pptxutil.ImportWarning{
		{Slide: 1, Message: "The shadow on 1 shape may look slightly different."},
		{Slide: 2, Message: "The shadow on 1 shape may look slightly different."},
	}
	if got := result.Warnings; !reflect.DeepEqual(got, want) {
		t.Errorf("warnings = %v, want %v", got, want)
	}
}

func TestImportWarnsThatEmbeddedFontsWereNotCarriedOver(t *testing.T) {
	for _, tc := range []struct {
		typefaces []string
		want      string
	}{
		{[]string{"Montserrat"}, "Embedded fonts (Montserrat) were not carried over, so text may look different."},
		{[]string{"Montserrat", "Lato", "Montserrat"},
			"Embedded fonts (Montserrat and Lato) were not carried over, so text may look different."},
		{[]string{"Montserrat", "Lato", "Open Sans"},
			"Embedded fonts (Montserrat, Lato and Open Sans) were not carried over, so text may look different."},
		{[]string{""}, "Embedded fonts were not carried over, so text may look different."},
	} {
		b := deck(``)
		embedFonts(b, tc.typefaces...)
		_, result := importPptx(t, b.zip(t), nil)
		want := []pptxutil.ImportWarning{{Slide: 0, Message: tc.want}}
		if got := result.Warnings; !reflect.DeepEqual(got, want) {
			t.Errorf("%q: warnings = %v, want %v", tc.typefaces, got, want)
		}
	}
}

func TestImportWithoutShadowsOrEmbeddedFontsHasNoSuchWarning(t *testing.T) {
	b := deck(shadowedShape("2", ``))
	b["ppt/presentation.xml"] = strings.Replace(b["ppt/presentation.xml"], `</p:presentation>`,
		`<p:embeddedFontLst/></p:presentation>`, 1)
	_, result := importPptx(t, b.zip(t), nil)
	if len(result.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", result.Warnings)
	}
}
