package pptxutil_test

// cspell:words algn Lato Montserrat prst Shdw srgb xfrm

import (
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
	want := []pptxutil.ImportWarning{
		{Slide: 1, Message: "Shadows on 5 shapes were left out."},
		{Slide: 3, Message: "Shadows on 5 shapes were left out."},
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
		{Slide: 1, Message: "The shadow on 1 shape was left out."},
		{Slide: 2, Message: "The shadow on 1 shape was left out."},
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
