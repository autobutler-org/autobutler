package odputil

// cspell:ignore xlink xmlns smil Calibri

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

// The namespace declarations every fixture part carries.
const xmlns = `xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" ` +
	`xmlns:style="urn:oasis:names:tc:opendocument:xmlns:style:1.0" ` +
	`xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0" ` +
	`xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0" ` +
	`xmlns:draw="urn:oasis:names:tc:opendocument:xmlns:drawing:1.0" ` +
	`xmlns:fo="urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0" ` +
	`xmlns:xlink="http://www.w3.org/1999/xlink" ` +
	`xmlns:dc="http://purl.org/dc/elements/1.1/" ` +
	`xmlns:meta="urn:oasis:names:tc:opendocument:xmlns:meta:1.0" ` +
	`xmlns:presentation="urn:oasis:names:tc:opendocument:xmlns:presentation:1.0" ` +
	`xmlns:svg="urn:oasis:names:tc:opendocument:xmlns:svg-compatible:1.0" ` +
	`xmlns:smil="urn:oasis:names:tc:opendocument:xmlns:smil-compatible:1.0" ` +
	`xmlns:anim="urn:oasis:names:tc:opendocument:xmlns:animation:1.0"`

// pngHeader is enough of a PNG for http.DetectContentType.
const pngHeader = "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"

// entry is one part of a test package.
type entry struct {
	name, body string
}

// fixture is a test presentation, built into a package by bytes.
type fixture struct {
	mime     string
	manifest string
	styles   string
	content  string
	extra    []entry
}

// defaultStyles is a styles.xml with a 28 cm × 15.75 cm master page, a
// title style and a font.
var defaultStyles = stylesXML(
	`<office:font-face-decls><style:font-face style:name="Liberation Sans" svg:font-family="'Liberation Sans'"/></office:font-face-decls>`+
		`<office:styles>`+
		`<style:default-style style:family="graphic"><style:graphic-properties draw:stroke="none" draw:fill="none"/></style:default-style>`+
		`<style:style style:name="Default-title" style:family="presentation">`+
		`<style:text-properties fo:font-size="44pt" style:font-name="Liberation Sans" fo:color="#112233"/></style:style>`+
		`<style:style style:name="Default-outline1" style:family="presentation">`+
		`<style:text-properties fo:font-size="32pt"/></style:style>`+
		`</office:styles>`,
	`<style:page-layout style:name="PM1"><style:page-layout-properties fo:page-width="28cm" fo:page-height="15.75cm"/></style:page-layout>`,
	`<style:master-page style:name="Default" style:page-layout-name="PM1"/>`,
)

func stylesXML(styles, automatic, masters string) string {
	return `<?xml version="1.0" encoding="UTF-8"?><office:document-styles ` + xmlns + `>` + styles +
		`<office:automatic-styles>` + automatic + `</office:automatic-styles>` +
		`<office:master-styles>` + masters + `</office:master-styles></office:document-styles>`
}

func contentXML(automatic, pages string) string {
	return `<?xml version="1.0" encoding="UTF-8"?><office:document-content ` + xmlns + `>` +
		`<office:automatic-styles>` + automatic + `</office:automatic-styles>` +
		`<office:body><office:presentation>` + pages + `</office:presentation></office:body></office:document-content>`
}

func page(inner string) string {
	return `<draw:page draw:name="p" draw:master-page-name="Default">` + inner + `</draw:page>`
}

func (f fixture) bytes(t testing.TB) []byte {
	t.Helper()
	if f.mime == "" {
		f.mime = mimePresentation
	}
	if f.styles == "" {
		f.styles = defaultStyles
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	parts := []entry{
		{name: "mimetype", body: f.mime},
		{name: "META-INF/manifest.xml", body: `<?xml version="1.0" encoding="UTF-8"?>` +
			`<manifest:manifest xmlns:manifest="urn:oasis:names:tc:opendocument:xmlns:manifest:1.0">` +
			`<manifest:file-entry manifest:full-path="/" manifest:media-type="` + f.mime + `"/>` +
			f.manifest + `</manifest:manifest>`},
		{name: "styles.xml", body: f.styles},
		{name: "content.xml", body: f.content},
	}
	for _, p := range append(parts, f.extra...) {
		w, err := zw.Create(p.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(w, p.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// deck is the .qslide decoded back for assertions.
type deck struct {
	SchemaVersion int            `json:"schemaVersion"`
	Title         string         `json:"title"`
	Size          outSize        `json:"size"`
	Slides        []decodedSlide `json:"slides"`
}

type decodedSlide struct {
	ID         string           `json:"id"`
	Background *outBackground   `json:"background"`
	Elements   []map[string]any `json:"elements"`
	Notes      string           `json:"notes"`
	Transition *outTransition   `json:"transition"`
}

// stored records what StoreMedia was handed.
type stored struct {
	part, name string
	size       int
}

func importFixture(t *testing.T, f fixture, store StoreMediaFunc) (deck, ImportOdpResult) {
	t.Helper()
	data := f.bytes(t)
	var out bytes.Buffer
	result, err := ImportOdp(ImportOdpParams{Source: bytes.NewReader(data), Size: int64(len(data)), Out: &out, StoreMedia: store, Title: "fallback"})
	if err != nil {
		t.Fatalf("ImportOdp: %v", err)
	}
	var d deck
	if err := json.Unmarshal(out.Bytes(), &d); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out.String())
	}
	if len(d.Slides) != result.Slides {
		t.Fatalf("wrote %d slides, result says %d", len(d.Slides), result.Slides)
	}
	return d, result
}

func importErr(t *testing.T, data []byte) error {
	t.Helper()
	_, err := ImportOdp(ImportOdpParams{Source: bytes.NewReader(data), Size: int64(len(data)), Out: io.Discard})
	return err
}

func recordStore(into *[]stored) StoreMediaFunc {
	return func(part, name string, r io.Reader) (string, error) {
		n, err := io.Copy(io.Discard, r)
		if err != nil {
			return "", err
		}
		*into = append(*into, stored{part: part, name: name, size: int(n)})
		return "part:" + part, nil
	}
}

// at reads a value out of decoded JSON: keys and indexes, in order.
func at(t *testing.T, v any, keys ...any) any {
	t.Helper()
	for _, k := range keys {
		switch k := k.(type) {
		case string:
			m, ok := v.(map[string]any)
			if !ok {
				t.Fatalf("at %q: not an object: %v", k, v)
			}
			v = m[k]
		case int:
			a, ok := v.([]any)
			if !ok || k >= len(a) {
				t.Fatalf("at %d: not an array that long: %v", k, v)
			}
			v = a[k]
		}
	}
	return v
}

func wantFrame(t *testing.T, el map[string]any, x, y, w, h float64) {
	t.Helper()
	frame := el["frame"].(map[string]any)
	got := [4]float64{frame["x"].(float64), frame["y"].(float64), frame["width"].(float64), frame["height"].(float64)}
	if got != [4]float64{x, y, w, h} {
		t.Fatalf("frame = %v, want %v", got, [4]float64{x, y, w, h})
	}
}

func wantWarning(t *testing.T, result ImportOdpResult, slide int, fragment string) {
	t.Helper()
	for _, w := range result.Warnings {
		if w.Slide == slide && strings.Contains(w.Message, fragment) {
			return
		}
	}
	t.Fatalf("no warning on slide %d containing %q in %+v", slide, fragment, result.Warnings)
}

func TestImportOdpLayout(t *testing.T) {
	d, result := importFixture(t, fixture{
		content: contentXML("", page(
			`<draw:frame presentation:class="title" presentation:style-name="Default-title" svg:x="2cm" svg:y="1cm" svg:width="24cm" svg:height="3cm">`+
				`<draw:text-box><text:p>Hello</text:p></draw:text-box></draw:frame>`)+
			page(`<draw:frame presentation:class="title" presentation:placeholder="true" svg:x="2cm" svg:y="1cm" svg:width="24cm" svg:height="3cm"><draw:text-box/></draw:frame>`)),
		extra: []entry{{name: "meta.xml", body: `<office:document-meta ` + xmlns + `><office:meta><dc:title> Quarterly </dc:title></office:meta></office:document-meta>`}},
	}, nil)
	if d.SchemaVersion != 1 || d.Title != "Quarterly" || d.Size != (outSize{Width: 1920, Height: 1080}) {
		t.Fatalf("header = %d %q %v", d.SchemaVersion, d.Title, d.Size)
	}
	if result.Slides != 2 || d.Slides[0].ID != "s1" || d.Slides[1].ID != "s2" {
		t.Fatalf("slides = %d, ids %q %q", result.Slides, d.Slides[0].ID, d.Slides[1].ID)
	}
	title := d.Slides[0].Elements[0]
	if title["type"] != "text" {
		t.Fatalf("title = %v", title)
	}
	// 1920 units over 28 cm: 68.57 units a centimeter.
	wantFrame(t, title, 137.14, 68.57, 1645.71, 205.71)
	run := at(t, title, "paragraphs", 0, "runs", 0).(map[string]any)
	// 44 pt is 1.552 cm.
	if run["text"] != "Hello" || run["fontSize"] != 106.44 || run["fontFamily"] != "Liberation Sans" || run["color"] != "#112233" {
		t.Fatalf("run = %v", run)
	}
	if len(d.Slides[1].Elements) != 0 {
		t.Fatalf("an empty placeholder was drawn: %v", d.Slides[1].Elements)
	}
}

func TestImportOdpSizes(t *testing.T) {
	for _, tc := range []struct {
		width, height string
		want          outSize
	}{
		{"25.4cm", "19.05cm", outSize{Width: 1024, Height: 768}},
		{"20cm", "20cm", outSize{Width: 1920, Height: 1920}},
		{"0cm", "1cm", outSize{Width: 1920, Height: 1080}},
	} {
		styles := stylesXML("",
			`<style:page-layout style:name="PM1"><style:page-layout-properties fo:page-width="`+tc.width+`" fo:page-height="`+tc.height+`"/></style:page-layout>`,
			`<style:master-page style:name="Default" style:page-layout-name="PM1"/>`)
		d, _ := importFixture(t, fixture{styles: styles, content: contentXML("", page(""))}, nil)
		if d.Size != tc.want {
			t.Errorf("%s × %s: size = %v, want %v", tc.width, tc.height, d.Size, tc.want)
		}
		if d.Title != "fallback" {
			t.Errorf("title = %q, want the caller's fallback", d.Title)
		}
	}
}

func TestImportOdpTextStyles(t *testing.T) {
	automatic := `<style:style style:name="P1" style:family="paragraph"><style:paragraph-properties fo:text-align="center" fo:line-height="150%"/>` +
		`<style:text-properties fo:font-style="italic"/></style:style>` +
		`<style:style style:name="T1" style:family="text"><style:text-properties fo:font-weight="bold" style:text-underline-style="solid" fo:font-family="Calibri"/></style:style>` +
		`<style:style style:name="T2" style:family="text"><style:text-properties style:text-line-through-style="solid" fo:color="#ff0000"/></style:style>` +
		`<style:style style:name="P2" style:family="paragraph"><style:paragraph-properties text:enable-numbering="false"/></style:style>` +
		`<text:list-style style:name="L1"><text:list-level-style-bullet text:level="1" text:bullet-char="•"/>` +
		`<text:list-level-style-number text:level="2" style:num-format="1"/></text:list-style>`
	d, _ := importFixture(t, fixture{content: contentXML(automatic, page(
		`<draw:frame svg:x="0cm" svg:y="0cm" svg:width="10cm" svg:height="5cm"><draw:text-box>`+
			`<text:p text:style-name="P1">  a  <text:span text:style-name="T1">b</text:span>`+"\n"+` c<text:s text:c="3"/>d<text:tab/>e<text:line-break/>  f <text:span text:style-name="T2">g</text:span> </text:p>`+
			`<text:list text:style-name="L1"><text:list-item><text:p>one</text:p>`+
			`<text:list><text:list-item><text:p>two</text:p></text:list-item></text:list></text:list-item>`+
			`<text:list-item><text:p text:style-name="P2">plain</text:p></text:list-item>`+
			`<text:list-header><text:p>head</text:p></text:list-header></text:list>`+
			`</draw:text-box></draw:frame>`)),
	}, nil)
	box := d.Slides[0].Elements[0]
	paragraphs := box["paragraphs"].([]any)
	first := paragraphs[0].(map[string]any)
	if first["align"] != "center" || first["lineSpacing"] != 1.5 {
		t.Fatalf("paragraph = %v", first)
	}
	var texts []string
	for _, r := range first["runs"].([]any) {
		run := r.(map[string]any)
		texts = append(texts, run["text"].(string))
		if run["italic"] != true {
			t.Fatalf("run %q lost its paragraph's italic: %v", run["text"], run)
		}
	}
	if got := strings.Join(texts, "|"); got != "a |b| c   d\te\nf |g" {
		t.Fatalf("runs = %q", got)
	}
	bold := at(t, first, "runs", 1).(map[string]any)
	if bold["bold"] != true || bold["underline"] != true || bold["fontFamily"] != "Calibri" {
		t.Fatalf("span run = %v", bold)
	}
	strike := at(t, first, "runs", 3).(map[string]any)
	if strike["strikethrough"] != true || strike["color"] != "#FF0000" || strike["bold"] != nil {
		t.Fatalf("second span run = %v", strike)
	}
	lists := []string{}
	for _, p := range paragraphs[1:] {
		list, _ := p.(map[string]any)["list"].(string)
		lists = append(lists, list)
	}
	if got := strings.Join(lists, ","); got != "bullet,numbered,," {
		t.Fatalf("lists = %q", got)
	}
}

func TestImportOdpShapes(t *testing.T) {
	automatic := `<style:style style:name="gr1" style:family="graphic"><style:graphic-properties draw:fill="solid" draw:fill-color="#00ff00" draw:opacity="50%" ` +
		`draw:stroke="dash" svg:stroke-color="#0000ff" svg:stroke-width="0.1cm" draw:shadow="visible" draw:shadow-offset-x="0.1cm" draw:shadow-offset-y="0.1cm" draw:shadow-color="#000000"/></style:style>` +
		`<style:style style:name="gr2" style:family="graphic"><style:graphic-properties draw:stroke="solid" svg:stroke-color="#000000" draw:marker-end="Arrow"/></style:style>` +
		`<style:style style:name="gr3" style:family="graphic"><style:graphic-properties draw:fill="gradient" draw:fill-gradient-name="G"/></style:style>`
	d, result := importFixture(t, fixture{content: contentXML(automatic, page(
		`<draw:custom-shape draw:style-name="gr1" svg:x="1cm" svg:y="1cm" svg:width="4cm" svg:height="2cm"><text:p>inside</text:p>`+
			`<draw:enhanced-geometry draw:type="ellipse"/></draw:custom-shape>`+
			`<draw:custom-shape svg:width="4cm" svg:height="2cm" draw:transform="rotate (1.5707963267949) translate (5cm 10cm)">`+
			`<draw:enhanced-geometry draw:type="mso-spt100"/></draw:custom-shape>`+
			`<draw:custom-shape svg:x="0cm" svg:y="0cm" svg:width="6cm" svg:height="3cm"><draw:enhanced-geometry draw:type="round-rectangle"/></draw:custom-shape>`+
			`<draw:custom-shape svg:x="0cm" svg:y="0cm" svg:width="2cm" svg:height="2cm"><draw:enhanced-geometry draw:type="ooxml-rtTriangle"/></draw:custom-shape>`+
			`<draw:custom-shape svg:x="0cm" svg:y="0cm" svg:width="2cm" svg:height="2cm"><draw:enhanced-geometry draw:type="right-arrow" draw:mirror-horizontal="true"/></draw:custom-shape>`+
			`<draw:rect draw:style-name="gr3" draw:corner-radius="0.5cm" svg:x="0cm" svg:y="0cm" svg:width="2cm" svg:height="2cm"/>`+
			`<draw:polygon svg:x="0cm" svg:y="0cm" svg:width="2cm" svg:height="2cm" svg:viewBox="0 0 100 100" draw:points="0,100 50,0 100,100"/>`+
			`<draw:line draw:style-name="gr2" svg:x1="10cm" svg:y1="2cm" svg:x2="2cm" svg:y2="6cm"/>`+
			`<draw:g><draw:rect svg:x="1cm" svg:y="1cm" svg:width="2cm" svg:height="2cm"/><draw:rect svg:x="5cm" svg:y="3cm" svg:width="1cm" svg:height="1cm"/></draw:g>`+
			`<draw:path svg:d="M0 0"/>`)),
		extra: []entry{},
	}, nil)
	els := d.Slides[0].Elements
	ellipse := els[0]
	if ellipse["kind"] != "ellipse" || ellipse["fill"] != "#00FF0080" {
		t.Fatalf("ellipse = %v", ellipse)
	}
	if s := ellipse["stroke"].(map[string]any); s["color"] != "#0000FF" || s["width"] != 6.86 || s["dash"] != "dash" {
		t.Fatalf("stroke = %v", s)
	}
	if sh := ellipse["shadow"].(map[string]any); sh["color"] != "#000000" || sh["x"] != 6.86 {
		t.Fatalf("shadow = %v", sh)
	}
	if text := els[1]; text["type"] != "text" || text["anchor"] != "middle" || at(t, text, "paragraphs", 0, "runs", 0, "text") != "inside" {
		t.Fatalf("shape text = %v", text)
	}
	rotated := els[2]
	wantFrame(t, rotated, 274.29, 480, 274.29, 137.14)
	if rotated["kind"] != "rectangle" || at(t, rotated, "frame", "rotation") != 270.0 {
		t.Fatalf("rotated = %v", rotated)
	}
	wantWarning(t, result, 1, "(mso-spt100)")
	if r := els[3]; r["kind"] != "roundedRectangle" || r["cornerRadius"] != 34.28 {
		t.Fatalf("rounded = %v", r)
	}
	if p := els[4]; p["kind"] != "path" || p["preset"] != "rtTriangle" {
		t.Fatalf("preset = %v", p)
	}
	if a := els[5]; a["kind"] != "arrow" || at(t, a, "frame", "rotation") != 180.0 {
		t.Fatalf("mirrored arrow = %v", a)
	}
	if r := els[6]; r["kind"] != "roundedRectangle" || r["cornerRadius"] != 34.29 {
		t.Fatalf("rect = %v", r)
	}
	wantWarning(t, result, 1, "Gradient fills")
	poly := els[7]
	if poly["kind"] != "path" || len(at(t, poly, "contours", 0, "commands").([]any)) != 4 ||
		at(t, poly, "contours", 0, "commands", 1, "x") != 0.5 {
		t.Fatalf("polygon = %v", poly)
	}
	line := els[8]
	wantFrame(t, line, 137.14, 137.14, 548.57, 274.29)
	if line["type"] != "line" || line["flipped"] != true || line["startCap"] != "arrow" || line["endCap"] != nil {
		t.Fatalf("line = %v", line)
	}
	group := els[9]
	wantFrame(t, group, 68.57, 68.57, 342.86, 205.71)
	wantFrame(t, at(t, group, "children", 1).(map[string]any), 274.29, 137.14, 68.57, 68.57)
	if len(els) != 10 {
		t.Fatalf("elements = %d, want 10", len(els))
	}
	wantWarning(t, result, 1, "Freeform drawings")
}

func TestImportOdpNotesTransitionsAndWarnings(t *testing.T) {
	automatic := `<style:style style:name="dp1" style:family="drawing-page"><style:drawing-page-properties draw:fill="solid" draw:fill-color="#abcdef" ` +
		`smil:type="barWipe" smil:subtype="topToBottom" smil:direction="reverse" smil:dur="1.5s"/></style:style>` +
		`<style:style style:name="dp2" style:family="drawing-page"><style:drawing-page-properties smil:type="fade" presentation:transition-speed="slow"/></style:style>` +
		`<style:style style:name="dp3" style:family="drawing-page"><style:drawing-page-properties smil:type="clockWipe"/></style:style>`
	d, result := importFixture(t, fixture{content: contentXML(automatic,
		`<draw:page draw:style-name="dp1">`+
			`<draw:frame svg:x="0cm" svg:y="0cm" svg:width="4cm" svg:height="4cm"><draw:object xlink:href="./Object 1"/><draw:image xlink:href="./ObjectReplacements/Object 1"/></draw:frame>`+
			`<draw:frame svg:x="0cm" svg:y="0cm" svg:width="4cm" svg:height="4cm"><draw:plugin xlink:href="movie.mp4"/></draw:frame>`+
			`<anim:par/>`+
			`<presentation:notes><draw:page-thumbnail/><draw:frame presentation:class="notes" svg:x="0cm" svg:y="0cm" svg:width="4cm" svg:height="4cm">`+
			`<draw:text-box><text:p>Say hi</text:p><text:p>then <text:span>go</text:span></text:p></draw:text-box></draw:frame></presentation:notes>`+
			`</draw:page>`+
			`<draw:page draw:style-name="dp2"/><draw:page draw:style-name="dp3"/>`)}, nil)
	first := d.Slides[0]
	if first.Notes != "Say hi\nthen go" {
		t.Fatalf("notes = %q", first.Notes)
	}
	if len(first.Elements) != 0 {
		t.Fatalf("elements = %v", first.Elements)
	}
	if first.Background == nil || first.Background.Color != "#ABCDEF" {
		t.Fatalf("background = %+v", first.Background)
	}
	if tr := first.Transition; tr == nil || *tr != (outTransition{Kind: "wipe", Direction: "up", Duration: 1500}) {
		t.Fatalf("transition = %+v", tr)
	}
	if tr := d.Slides[1].Transition; tr == nil || *tr != (outTransition{Kind: "fade", Duration: 1000}) {
		t.Fatalf("fade = %+v", tr)
	}
	if d.Slides[2].Transition != nil {
		t.Fatalf("a clock wipe came across: %+v", d.Slides[2].Transition)
	}
	wantWarning(t, result, 1, "Charts and embedded objects")
	wantWarning(t, result, 1, "Video and audio")
	wantWarning(t, result, 1, "Animations")
	wantWarning(t, result, 3, "transition has no match")
}

func TestImportOdpPictures(t *testing.T) {
	png := pngHeader + strings.Repeat("x", 600)
	image := func(alt string) string {
		return `<draw:frame svg:x="1cm" svg:y="1cm" svg:width="4cm" svg:height="3cm">` +
			`<draw:image xlink:href="Pictures/logo.svg"/><draw:image xlink:href="Pictures/logo.png"/>` +
			`<svg:title>` + alt + `</svg:title></draw:frame>`
	}
	f := fixture{
		content: contentXML("", page(image("Logo")+
			`<draw:frame svg:x="0cm" svg:y="0cm" svg:width="1cm" svg:height="1cm"><draw:image xlink:href="https://example.com/a.png"/></draw:frame>`+
			`<draw:frame svg:x="0cm" svg:y="0cm" svg:width="1cm" svg:height="1cm"><draw:image xlink:href="Pictures/gone.png"/></draw:frame>`)+
			page(image("Again"))),
		extra: []entry{
			{name: "Pictures/logo.svg", body: `<svg xmlns="http://www.w3.org/2000/svg"/>`},
			{name: "Pictures/logo.png", body: png},
		},
	}
	var got []stored
	d, result := importFixture(t, f, recordStore(&got))
	if len(got) != 1 || got[0] != (stored{part: "Pictures/logo.png", name: "logo.png", size: len(png)}) {
		t.Fatalf("stored = %+v", got)
	}
	if result.Pictures != 1 {
		t.Fatalf("pictures = %d", result.Pictures)
	}
	for i, alt := range []string{"Logo", "Again"} {
		el := d.Slides[i].Elements[0]
		if el["type"] != "image" || el["source"] != "part:Pictures/logo.png" || el["altText"] != alt || el["fit"] != "fill" {
			t.Fatalf("slide %d picture = %v", i+1, el)
		}
	}
	if len(d.Slides[0].Elements) != 1 {
		t.Fatalf("linked or missing pictures were drawn: %v", d.Slides[0].Elements)
	}
	wantWarning(t, result, 1, "Linked pictures")
	wantWarning(t, result, 1, "missing from the file")
	for _, w := range result.Warnings {
		if strings.Contains(w.Message, "SVG") {
			t.Fatalf("the SVG with a PNG beside it was warned about: %+v", w)
		}
	}

	_, result = importFixture(t, f, nil)
	wantWarning(t, result, 1, "Pictures are not imported here.")
}

func TestImportOdpBackgroundsAndMasters(t *testing.T) {
	styles := stylesXML(
		`<office:styles><draw:fill-image draw:name="Paper" xlink:href="Pictures/paper.png"/>`+
			`<style:style style:name="Default-title" style:family="presentation"/></office:styles>`,
		`<style:page-layout style:name="PM1"><style:page-layout-properties fo:page-width="28cm" fo:page-height="15.75cm"/></style:page-layout>`+
			`<style:style style:name="Mdp1" style:family="drawing-page"><style:drawing-page-properties draw:fill="solid" draw:fill-color="#010203"/></style:style>`+
			`<style:style style:name="gr1" style:family="graphic"><style:graphic-properties draw:fill="solid" draw:fill-color="#ff00ff"/></style:style>`,
		`<style:master-page style:name="Default" style:page-layout-name="PM1" draw:style-name="Mdp1">`+
			`<draw:frame presentation:class="title" svg:x="3cm" svg:y="2cm" svg:width="20cm" svg:height="2cm"><draw:text-box/></draw:frame>`+
			`<draw:custom-shape draw:style-name="gr1" svg:x="0cm" svg:y="0cm" svg:width="28cm" svg:height="1cm"><draw:enhanced-geometry draw:type="rectangle"/></draw:custom-shape>`+
			`</style:master-page>`)
	automatic := `<style:style style:name="dp1" style:family="drawing-page"><style:drawing-page-properties draw:fill="bitmap" draw:fill-image-name="Paper"/></style:style>` +
		`<style:style style:name="dp2" style:family="drawing-page"><style:drawing-page-properties presentation:background-objects-visible="false"/></style:style>` +
		// The same name as a master's automatic style, which is a different style.
		`<style:style style:name="gr1" style:family="graphic"><style:graphic-properties draw:fill="solid" draw:fill-color="#00ffff"/></style:style>`
	var got []stored
	d, _ := importFixture(t, fixture{
		styles: styles,
		content: contentXML(automatic,
			`<draw:page draw:style-name="dp1" draw:master-page-name="Default">`+
				`<draw:frame presentation:class="title"><draw:text-box><text:p>Placed by the master</text:p></draw:text-box></draw:frame></draw:page>`+
				`<draw:page draw:style-name="dp2" draw:master-page-name="Default"/>`),
		extra: []entry{{name: "Pictures/paper.png", body: pngHeader}},
	}, recordStore(&got))
	first := d.Slides[0]
	if first.Background == nil || first.Background.Image != "part:Pictures/paper.png" {
		t.Fatalf("background = %+v", first.Background)
	}
	if len(first.Elements) != 2 {
		t.Fatalf("elements = %v", first.Elements)
	}
	if band := first.Elements[0]; band["fill"] != "#FF00FF" {
		t.Fatalf("the master's shape = %v", band)
	}
	wantFrame(t, first.Elements[1], 205.71, 137.14, 1371.43, 137.14)
	second := d.Slides[1]
	if second.Background == nil || second.Background.Color != "#010203" {
		t.Fatalf("the master's background = %+v", second.Background)
	}
	if len(second.Elements) != 0 {
		t.Fatalf("the master's shapes were drawn on a slide that hides them: %v", second.Elements)
	}
}

func TestImportOdpTable(t *testing.T) {
	automatic := `<style:style style:name="co1" style:family="table-column"><style:table-column-properties style:column-width="4cm"/></style:style>` +
		`<style:style style:name="ro1" style:family="table-row"><style:table-row-properties style:row-height="1cm"/></style:style>` +
		`<style:style style:name="ce1" style:family="table-cell"><style:graphic-properties draw:fill="solid" draw:fill-color="#eeeeee" draw:textarea-vertical-align="middle"/>` +
		`<style:paragraph-properties fo:border="0.03cm solid #333333" fo:border-left="none"/></style:style>`
	d, result := importFixture(t, fixture{content: contentXML(automatic, page(
		`<draw:frame svg:x="1cm" svg:y="1cm" svg:width="10cm" svg:height="3cm"><table:table table:use-first-row-styles="true">`+
			`<table:table-column table:style-name="co1"/><table:table-column/>`+
			`<table:table-row table:style-name="ro1"><table:table-cell table:style-name="ce1" table:number-columns-spanned="2"><text:p>wide</text:p></table:table-cell><table:covered-table-cell/></table:table-row>`+
			`<table:table-row table:number-rows-repeated="2"><table:table-cell><text:p>a</text:p></table:table-cell><table:table-cell/></table:table-row>`+
			`</table:table></draw:frame>`+
			`<draw:frame svg:x="1cm" svg:y="1cm" svg:width="10cm" svg:height="3cm"><table:table>`+
			`<table:table-row table:number-rows-repeated="1000000000"><table:table-cell table:number-columns-repeated="1000000000"/></table:table-row>`+
			`</table:table></draw:frame>`)),
	}, nil)
	els := d.Slides[0].Elements
	if len(els) != 1 {
		t.Fatalf("elements = %v", els)
	}
	tbl := els[0]
	if tbl["type"] != "table" || tbl["headerRow"] != true {
		t.Fatalf("table = %v", tbl)
	}
	// 4 cm, then the rest of the 10 cm frame; 1 cm, then the 2 cm left over
	// shared by two.
	if fmt.Sprint(tbl["columns"]) != "[274.29 411.42]" || fmt.Sprint(tbl["rows"]) != "[68.57 68.57 68.57]" {
		t.Fatalf("columns %v rows %v", tbl["columns"], tbl["rows"])
	}
	wantFrame(t, tbl, 68.57, 68.57, 685.71, 205.71)
	cell := at(t, tbl, "cells", 0, 0).(map[string]any)
	if cell["colSpan"] != 2.0 || cell["fill"] != "#EEEEEE" || cell["anchor"] != "middle" ||
		at(t, cell, "paragraphs", 0, "runs", 0, "text") != "wide" {
		t.Fatalf("cell = %v", cell)
	}
	borders := cell["borders"].(map[string]any)
	if borders["left"] != nil || at(t, borders, "top", "color") != "#333333" || at(t, borders, "top", "width") != 2.06 {
		t.Fatalf("borders = %v", borders)
	}
	if at(t, tbl, "cells", 2, 0, "paragraphs", 0, "runs", 0, "text") != "a" {
		t.Fatalf("repeated row = %v", at(t, tbl, "cells", 2))
	}
	wantWarning(t, result, 1, "Tables of more than")
}

func TestImportOdpMalformed(t *testing.T) {
	good := contentXML("", page(""))
	for _, tc := range []struct {
		name string
		data []byte
		want error
	}{
		{"not a zip", []byte("this is not a zip at all"), ErrMalformed},
		{"a text document", fixture{mime: "application/vnd.oasis.opendocument.text", content: good}.bytes(t), ErrMalformed},
		{"no presentation", fixture{content: strings.Replace(good, "office:presentation", "office:text", 2)}.bytes(t), ErrMalformed},
		{"broken xml", fixture{content: good[:len(good)-30]}.bytes(t), ErrMalformed},
		{"broken styles", fixture{styles: "<office:document-styles", content: good}.bytes(t), ErrMalformed},
		{"encrypted", fixture{
			manifest: `<manifest:file-entry manifest:full-path="content.xml" manifest:media-type="text/xml"><manifest:encryption-data/></manifest:file-entry>`,
			content:  good,
		}.bytes(t), ErrEncrypted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := importErr(t, tc.data); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if _, err := zw.Create("mimetype"); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := importErr(t, buf.Bytes()); !errors.Is(err, ErrMalformed) {
		t.Fatalf("a package without content.xml: err = %v", err)
	}
	if _, err := ImportOdp(ImportOdpParams{Source: bytes.NewReader(nil), Out: io.Discard}); !errors.Is(err, ErrMalformed) {
		t.Fatalf("an empty file: err = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	data := fixture{content: good}.bytes(t)
	if _, err := ImportOdp(ImportOdpParams{Ctx: ctx, Source: bytes.NewReader(data), Size: int64(len(data)), Out: io.Discard}); !errors.Is(err, context.Canceled) {
		t.Fatalf("a canceled import: err = %v", err)
	}
}

func TestImportOdpOversize(t *testing.T) {
	t.Run("too many slides", func(t *testing.T) {
		pages := strings.Repeat(`<draw:page/>`, MaxSlides+1)
		if err := importErr(t, fixture{content: contentXML("", pages)}.bytes(t)); !errors.Is(err, ErrTooLarge) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("a slide past its node budget", func(t *testing.T) {
		shapes := strings.Repeat(`<text:span/>`, maxPageNodes)
		content := contentXML("", page(`<draw:frame svg:width="1cm" svg:height="1cm"><draw:text-box><text:p>`+shapes+`</text:p></draw:text-box></draw:frame>`))
		if err := importErr(t, fixture{content: content}.bytes(t)); !errors.Is(err, ErrTooLarge) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("spaces that expand past the text cap", func(t *testing.T) {
		spaces := strings.Repeat(`<text:s text:c="10000"/>x`, maxSlideTextBytes/maxSpaceRun+1)
		content := contentXML("", page(`<draw:frame svg:width="1cm" svg:height="1cm"><draw:text-box><text:p>`+spaces+`</text:p></draw:text-box></draw:frame>`))
		if err := importErr(t, fixture{content: content}.bytes(t)); !errors.Is(err, ErrTooLarge) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("groups nested too deep", func(t *testing.T) {
		deep := strings.Repeat(`<draw:g>`, MaxGroupDepth+2) + `<draw:rect svg:width="1cm" svg:height="1cm"/>` + strings.Repeat(`</draw:g>`, MaxGroupDepth+2)
		if err := importErr(t, fixture{content: contentXML("", page(deep))}.bytes(t)); !errors.Is(err, ErrTooLarge) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("a style cycle", func(t *testing.T) {
		automatic := `<style:style style:name="a" style:family="graphic" style:parent-style-name="b"/>`
		styles := stylesXML(`<office:styles><style:style style:name="b" style:family="graphic" style:parent-style-name="b"/></office:styles>`, "", "")
		d, _ := importFixture(t, fixture{styles: styles, content: contentXML(automatic, page(`<draw:rect draw:style-name="a" svg:width="1cm" svg:height="1cm"/>`))}, nil)
		if len(d.Slides[0].Elements) != 1 {
			t.Fatalf("elements = %v", d.Slides[0].Elements)
		}
	})
}

// bombPackage is a presentation whose named part unpacks to size bytes of
// zeros behind prefix, deflated to almost nothing.
func bombPackage(t *testing.T, name, prefix string, size int64, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, p := range []entry{{name: "mimetype", body: mimePresentation}, {name: "styles.xml", body: defaultStyles}} {
		w, err := zw.Create(p.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(w, p.body); err != nil {
			t.Fatal(err)
		}
	}
	w, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, prefix); err != nil {
		t.Fatal(err)
	}
	if _, err := io.CopyN(w, zeros{}, size); err != nil {
		t.Fatal(err)
	}
	if content != "" {
		w, err := zw.Create(contentPart)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(w, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// zeros reads as an endless run of zero bytes.
type zeros struct{}

func (zeros) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

func TestImportOdpZipBomb(t *testing.T) {
	if testing.Short() {
		t.Skip("unpacks past the part limit")
	}
	t.Run("content.xml", func(t *testing.T) {
		data := bombPackage(t, contentPart, `<?xml version="1.0"?><office:document-content `+xmlns+`><!--`, MaxPartBytes, "")
		if len(data) > 1<<20 {
			t.Fatalf("the bomb is %d bytes; it should be small", len(data))
		}
		if err := importErr(t, data); !errors.Is(err, ErrTooLarge) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("a picture", func(t *testing.T) {
		content := contentXML("", page(`<draw:frame svg:width="1cm" svg:height="1cm"><draw:image xlink:href="Pictures/huge.png"/></draw:frame>`))
		data := bombPackage(t, "Pictures/huge.png", pngHeader, MaxPartBytes, content)
		var got []stored
		result, err := ImportOdp(ImportOdpParams{Source: bytes.NewReader(data), Size: int64(len(data)), Out: io.Discard, StoreMedia: recordStore(&got)})
		if err != nil {
			t.Fatalf("a picture past the limit failed the import: %v", err)
		}
		if len(got) != 0 || result.Pictures != 0 {
			t.Fatalf("the oversize picture was stored: %+v", got)
		}
		wantWarning(t, result, 1, "Pictures larger than")
	})
}
