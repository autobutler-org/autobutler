package odputil_test

// cspell:ignore alink crossfade smil xlink Impress

import (
	"bytes"
	"errors"
	"io"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/odputil"
)

// unitsPerCm is how many slide units a cm is on the default 28 cm wide
// page, which becomes 1920 units.
const unitsPerCm = 1920.0 / 28

// TestImportLibreOfficeSample reads a deck Impress itself saved: a title
// slide, a slide with an outline, a rotated rectangle holding text, a
// circle, an arrow, speaker notes and a wipe, and a slide with a picture,
// a table and its own background.
func TestImportLibreOfficeSample(t *testing.T) {
	data, err := os.ReadFile("testdata/sample.odp")
	if err != nil {
		t.Fatal(err)
	}
	store := &storeAll{}
	d, res, err := importWith(t, data, store.store)
	if err != nil {
		t.Fatalf("ImportOdp: %v", err)
	}
	if d.SchemaVersion != 1 || d.Title != "Sample deck" || d.Size != (size{1920, 1080}) {
		t.Fatalf("header = %d %q %v", d.SchemaVersion, d.Title, d.Size)
	}
	if res.Slides != 3 || len(d.Slides) != 3 || res.Pictures != 1 || len(res.Warnings) != 0 {
		t.Fatalf("result = %+v", res)
	}

	title := d.Slides[0].Elements[0]
	if title.Type != "text" || texts(title.Paragraphs) != "Hello Impress" || title.Paragraphs[0].Align != "center" ||
		title.Anchor != "middle" {
		t.Errorf("title = %+v", title)
	}
	if r := title.Paragraphs[0].Runs[0]; !near(*r.FontSize, 44*2.54/72*unitsPerCm, 0.01) || *r.FontFamily != "Liberation Sans" {
		t.Errorf("title run = %v %v", *r.FontSize, *r.FontFamily)
	}

	s2 := d.Slides[1].Elements
	outline := s2[1]
	if texts(outline.Paragraphs) != "First point\nSecond point" || outline.Paragraphs[0].List != "bullet" ||
		outline.AutoFit != "shrink" {
		t.Errorf("outline = %+v", outline)
	}
	rect, label := s2[2], s2[3]
	if rect.Type != "shape" || rect.Kind != "rectangle" || rect.Fill != "#00FF00" || rect.Frame.Rotation != 330 {
		t.Errorf("rotated rectangle = %+v", rect)
	}
	// Impress placed it at 2 cm, 10 cm, 4 cm by 2 cm, then turned it 30°
	// about its center.
	if !near(rect.Frame.X, 2*unitsPerCm, 0.1) || !near(rect.Frame.Y, 10*unitsPerCm, 0.1) ||
		!near(rect.Frame.Width, 4*unitsPerCm, 0.1) {
		t.Errorf("rotated rectangle frame = %+v", rect.Frame)
	}
	if label.Type != "text" || texts(label.Paragraphs) != "Boxed" || label.Frame.Rotation != 330 || label.Anchor != "middle" {
		t.Errorf("rectangle text = %+v", label)
	}
	if circle := s2[4]; circle.Kind != "ellipse" || circle.Stroke == nil {
		t.Errorf("circle = %+v", circle)
	}
	if arrow := s2[5]; arrow.Type != "line" || arrow.EndCap != "arrow" || arrow.StartCap != "" || arrow.Flipped {
		t.Errorf("arrow = %+v", arrow)
	}
	if d.Slides[1].Notes != "Speaker note one\nline two" {
		t.Errorf("notes = %q", d.Slides[1].Notes)
	}
	if tr := d.Slides[1].Transition; tr == nil || *tr != (transition{"wipe", "right", 2000}) {
		t.Errorf("transition = %+v", tr)
	}

	s3 := d.Slides[2]
	if s3.Background == nil || s3.Background.Color != "#123456" {
		t.Errorf("background = %+v", s3.Background)
	}
	img := s3.Elements[0]
	if img.Type != "image" || img.AltText != "Red box" || !strings.HasPrefix(img.Source, "media/") ||
		!strings.HasSuffix(img.Source, ".png") {
		t.Errorf("picture = %+v", img)
	}
	tbl := s3.Elements[1]
	if tbl.Type != "table" || len(tbl.Rows) != 2 || len(tbl.Columns) != 2 || texts(tbl.Cells[1][1].Paragraphs) != "r1c1" {
		t.Errorf("table = %+v", tbl)
	}
	if store.calls != 1 {
		t.Errorf("StoreMedia called %d times", store.calls)
	}
}

// TestImportEmptyDeck writes a deck with no slides as the codec does, and
// takes the caller's title when the package names none.
func TestImportEmptyDeck(t *testing.T) {
	var out bytes.Buffer
	data := fixture{}.build(t)
	res, err := odputil.ImportOdp(odputil.ImportOdpParams{Source: bytes.NewReader(data), Size: int64(len(data)), Out: &out, Title: "Mine"})
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"schemaVersion\": 1,\n  \"title\": \"Mine\",\n  \"size\": {\n    \"width\": 1920,\n    \"height\": 1080\n  },\n  \"slides\": []\n}\n"
	if out.String() != want || res.Slides != 0 {
		t.Fatalf("got %q", out.String())
	}
}

// TestImportSlideSize takes the size from the first master's page layout:
// 4:3 is the editor's 1024×768, and any other shape 1920 wide.
func TestImportSlideSize(t *testing.T) {
	for _, tc := range []struct {
		w, h string
		want size
	}{
		{"28cm", "21cm", size{1024, 768}},
		{"10in", "5in", size{1920, 960}},
		{"0cm", "15cm", size{1920, 1080}},
		{"NaNcm", "15cm", size{1920, 1080}},
	} {
		f := fixture{
			masterAuto: `<style:page-layout style:name="PM1"><style:page-layout-properties fo:page-width="` + tc.w +
				`" fo:page-height="` + tc.h + `"/></style:page-layout>`,
			masters: `<style:master-page style:name="M" style:page-layout-name="PM1"/>`,
		}
		if d, _ := mustImport(t, f); d.Size != tc.want {
			t.Errorf("%s×%s: size = %v, want %v", tc.w, tc.h, d.Size, tc.want)
		}
	}
}

// TestImportTextStyles resolves a run's style along its span's, its
// paragraph's and its frame's styles and their parents, a percentage size
// scaling the size beneath it, and a font named by its face.
func TestImportTextStyles(t *testing.T) {
	f := fixture{
		styles: `<style:default-style style:family="graphic"><style:text-properties fo:font-size="20pt"/></style:default-style>` +
			`<style:style style:name="Base" style:family="graphic"><style:text-properties fo:color="#112233" style:font-name="Face"/></style:style>`,
		autoStyles: `<style:style style:name="gr1" style:family="graphic" style:parent-style-name="Base"/>` +
			`<style:style style:name="P1" style:family="paragraph"><style:paragraph-properties fo:text-align="end" fo:line-height="150%"/></style:style>` +
			`<style:style style:name="T1" style:family="text"><style:text-properties fo:font-weight="bold" fo:font-style="italic" fo:font-size="200%"/></style:style>` +
			`<style:style style:name="T2" style:family="text"><style:text-properties style:text-underline-style="solid" style:text-line-through-style="solid" fo:font-family="'Mono Font', monospace"/></style:style>`,
		pages: page(textFrame(`draw:style-name="gr1"`,
			`<text:p text:style-name="P1">plain <text:span text:style-name="T1">big</text:span>`+
				`<text:span text:style-name="T2">marked</text:span></text:p>`)),
	}
	d, _ := mustImport(t, f)
	el := d.Slides[0].Elements[0]
	p := el.Paragraphs[0]
	if p.Align != "end" || p.LineSpacing == nil || *p.LineSpacing != 1.8 {
		t.Fatalf("paragraph = %+v", p)
	}
	if len(p.Runs) != 3 {
		t.Fatalf("runs = %+v", p.Runs)
	}
	plain, big, marked := p.Runs[0], p.Runs[1], p.Runs[2]
	pt := 2.54 / 72 * unitsPerCm
	if plain.Text != "plain " || plain.Color != "#112233" || *plain.FontFamily != "Face" || !near(*plain.FontSize, 20*pt, 0.01) {
		t.Errorf("plain = %+v %v %v", plain, *plain.FontSize, *plain.FontFamily)
	}
	if !big.Bold || !big.Italic || !near(*big.FontSize, 40*pt, 0.01) {
		t.Errorf("big = %+v %v", big, *big.FontSize)
	}
	if !marked.Underline || !marked.Strikethrough || *marked.FontFamily != "Mono Font" || marked.Bold {
		t.Errorf("marked = %+v", marked)
	}
}

// TestImportFontFaces maps a style:font-name to the family its font face
// declares.
func TestImportFontFaces(t *testing.T) {
	f := fixture{
		contentRaw: `<office:document-content ` + namespaces + `><office:font-face-decls>` +
			`<style:font-face style:name="F1" svg:font-family="'DejaVu Serif'"/></office:font-face-decls>` +
			`<office:automatic-styles><style:style style:name="T1" style:family="text"><style:text-properties style:font-name="F1"/></style:style></office:automatic-styles>` +
			`<office:body><office:presentation>` +
			page(textFrame("", `<text:p><text:span text:style-name="T1">x</text:span></text:p>`)) +
			`</office:presentation></office:body></office:document-content>`,
	}
	d, _ := mustImport(t, f)
	if got := d.Slides[0].Elements[0].Paragraphs[0].Runs[0].FontFamily; got == nil || *got != "DejaVu Serif" {
		t.Fatalf("font = %v", got)
	}
}

// TestImportWhiteSpace collapses white space as ODF does, and keeps the
// spaces, tabs and line breaks the file marks.
func TestImportWhiteSpace(t *testing.T) {
	f := fixture{pages: page(textFrame("",
		"<text:p>\n  one   two\t<text:s text:c=\"3\"/>three<text:tab/>four<text:line-break/>  five</text:p>"))}
	d, _ := mustImport(t, f)
	ps := d.Slides[0].Elements[0].Paragraphs
	if got := texts(ps); got != "one two    three\tfour\nfive" {
		t.Fatalf("text = %q", got)
	}
	if len(ps) != 2 || len(ps[0].Runs) != 1 {
		t.Fatalf("paragraphs = %+v", ps)
	}
}

// TestImportLeavesOutNotesAndAnnotations keeps a field's text and drops a
// note's body and a comment.
func TestImportLeavesOutNotesAndAnnotations(t *testing.T) {
	f := fixture{pages: page(textFrame("",
		`<text:p>a<text:note><text:note-body><text:p>hidden</text:p></text:note-body></text:note>`+
			`<office:annotation><text:p>comment</text:p></office:annotation>`+
			`<text:a xlink:href="https://example.com">link</text:a> <text:page-number>7</text:page-number></text:p>`))}
	d, _ := mustImport(t, f)
	if got := texts(d.Slides[0].Elements[0].Paragraphs); got != "alink 7" {
		t.Fatalf("text = %q", got)
	}
}

// TestImportLists marks list paragraphs by their list style's level: a
// bullet, a number, none for a number with no format, nothing for a list
// header, and a bullet in an outline placeholder outside any list.
func TestImportLists(t *testing.T) {
	f := fixture{
		styles: `<style:style style:name="Outline" style:family="presentation"><style:graphic-properties>` +
			`<text:list-style style:name="O"><text:list-level-style-number text:level="1" style:num-format="1"/></text:list-style>` +
			`</style:graphic-properties></style:style>`,
		autoStyles: `<text:list-style style:name="L1">` +
			`<text:list-level-style-bullet text:level="1" text:bullet-char="•"/>` +
			`<text:list-level-style-number text:level="2" style:num-format="1"/>` +
			`<text:list-level-style-number text:level="3" style:num-format=""/></text:list-style>`,
		pages: page(textFrame("",
			`<text:list text:style-name="L1"><text:list-item><text:p>a</text:p>`+
				`<text:list><text:list-item><text:p>b</text:p><text:list><text:list-item><text:p>c</text:p></text:list-item></text:list></text:list-item></text:list>`+
				`</text:list-item><text:list-header><text:p>h</text:p></text:list-header></text:list>`+
				`<text:list><text:list-item><text:p>u</text:p></text:list-item></text:list>`) +
			textFrame(`presentation:class="outline" presentation:style-name="Outline"`, `<text:p>o</text:p>`)),
	}
	d, _ := mustImport(t, f)
	var got []string
	for _, p := range d.Slides[0].Elements[0].Paragraphs {
		got = append(got, texts([]paragraph{p})+":"+p.List)
	}
	if want := []string{"a:bullet", "b:numbered", "c:", "h:", "u:bullet"}; !slices.Equal(got, want) {
		t.Errorf("lists = %v, want %v", got, want)
	}
	if p := d.Slides[0].Elements[1].Paragraphs[0]; p.List != "numbered" {
		t.Errorf("outline paragraph = %+v", p)
	}
}

// TestImportTextBoxLayout insets a text box by its padding and reads its
// anchor and auto-fit.
func TestImportTextBoxLayout(t *testing.T) {
	f := fixture{
		autoStyles: `<style:style style:name="g1" style:family="graphic"><style:graphic-properties fo:padding="1cm" ` +
			`fo:padding-left="2cm" draw:textarea-vertical-align="bottom" draw:auto-grow-height="false"/></style:style>` +
			`<style:style style:name="g2" style:family="graphic"><style:graphic-properties style:shrink-to-fit="true"/></style:style>`,
		pages: page(textFrame(`draw:style-name="g1"`, `<text:p>a</text:p>`) + textFrame(`draw:style-name="g2"`, `<text:p>b</text:p>`) +
			textFrame("", `<text:p>c</text:p>`)),
	}
	d, _ := mustImport(t, f)
	els := d.Slides[0].Elements
	box := els[0]
	if !near(box.Frame.X, 2*unitsPerCm, 0.01) || !near(box.Frame.Y, 1*unitsPerCm, 0.01) ||
		!near(box.Frame.Width, 7*unitsPerCm, 0.02) || !near(box.Frame.Height, 3*unitsPerCm, 0.02) {
		t.Errorf("inset frame = %+v", box.Frame)
	}
	if box.Anchor != "bottom" || box.AutoFit != "fixed" {
		t.Errorf("box = %q %q", box.Anchor, box.AutoFit)
	}
	if els[1].AutoFit != "shrink" || els[2].AutoFit != "" || els[2].Anchor != "" {
		t.Errorf("auto-fit = %q, %q %q", els[1].AutoFit, els[2].AutoFit, els[2].Anchor)
	}
}

// TestImportShapes maps custom shapes and the plain ODF shapes onto the
// editor's kinds, with their fill, outline and opacity, and warns for what
// becomes a rectangle.
func TestImportShapes(t *testing.T) {
	shape := func(typ, extra string) string {
		return `<draw:custom-shape draw:style-name="g1" svg:x="1cm" svg:y="1cm" svg:width="4cm" svg:height="2cm">` +
			`<draw:enhanced-geometry draw:type="` + typ + `" ` + extra + `/></draw:custom-shape>`
	}
	f := fixture{
		styles: `<draw:stroke-dash draw:name="Dots" draw:style="round" draw:dots1="1"/>` +
			`<draw:stroke-dash draw:name="DashDot" draw:dots1="1" draw:dots1-length="0.5cm" draw:dots2="1" draw:distance="0.2cm"/>` +
			`<draw:gradient draw:name="G" draw:start-color="#ABCDEF" draw:end-color="#000000"/>`,
		autoStyles: `<style:style style:name="g1" style:family="graphic"><style:graphic-properties draw:fill="solid" ` +
			`draw:fill-color="#ff0000" draw:stroke="dash" draw:stroke-dash="Dots" svg:stroke-color="#00ff00" ` +
			`svg:stroke-width="0.1cm" draw:opacity="50%"/></style:style>` +
			`<style:style style:name="g2" style:family="graphic"><style:graphic-properties draw:fill="gradient" ` +
			`draw:fill-gradient-name="G" draw:stroke="dash" draw:stroke-dash="DashDot"/></style:style>`,
		pages: page(shape("round-rectangle", `draw:modifiers="5400"`) + shape("star5", "") + shape("hexagon", "") +
			shape("non-primitive", "") + shape("ellipse", "") + shape("isosceles-triangle", "") +
			`<draw:rect draw:style-name="g2" svg:x="0cm" svg:y="0cm" svg:width="2cm" svg:height="2cm" draw:corner-radius="0.5cm"/>` +
			`<draw:ellipse svg:x="0cm" svg:y="0cm" svg:width="2cm" svg:height="2cm"/>` +
			`<draw:polygon svg:x="0cm" svg:y="0cm" svg:width="2cm" svg:height="2cm" draw:points="0,0 1,1"/>`),
	}
	d, res := mustImport(t, f)
	els := d.Slides[0].Elements
	var kinds []string
	for _, el := range els {
		kinds = append(kinds, el.Kind)
	}
	want := []string{"roundedRectangle", "star", "rectangle", "rectangle", "ellipse", "triangle", "roundedRectangle", "ellipse", "rectangle"}
	if !slices.Equal(kinds, want) {
		t.Fatalf("kinds = %v, want %v", kinds, want)
	}
	rr := els[0]
	// 5,400 of 21,600 is a quarter of the shorter side, 2 cm.
	if rr.CornerRadius == nil || !near(*rr.CornerRadius, 0.5*unitsPerCm, 0.01) {
		t.Errorf("corner radius = %v", rr.CornerRadius)
	}
	if rr.Fill != "#FF0000" || rr.Opacity == nil || *rr.Opacity != 0.5 || rr.Stroke == nil ||
		*rr.Stroke != (stroke{"#00FF00", 6.86, "dot"}) {
		t.Errorf("styled shape = %+v %+v", rr, rr.Stroke)
	}
	if r := els[6]; r.Fill != "#ABCDEF" || r.Stroke == nil || r.Stroke.Dash != "dashDot" || r.Stroke.Color != "#000000" ||
		r.CornerRadius == nil || !near(*r.CornerRadius, 0.5*unitsPerCm, 0.01) {
		t.Errorf("rect = %+v %+v", r, r.Stroke)
	}
	if plain := els[7]; plain.Fill != "" || plain.Stroke != nil {
		t.Errorf("unstyled ellipse = %+v", plain)
	}
	for _, w := range []string{
		"Shapes of a kind the editor does not have (hexagon) were drawn as rectangles.",
		"Freeform shapes were drawn as rectangles.",
		"Gradient and hatched fills were drawn in a single color.",
	} {
		if !slices.Contains(warnings(res), w) {
			t.Errorf("warnings %v lack %q", warnings(res), w)
		}
	}
}

// TestImportShapeText draws a shape's text as a text box over it,
// centered and inset by the shape's padding.
func TestImportShapeText(t *testing.T) {
	f := fixture{
		autoStyles: `<style:style style:name="g1" style:family="graphic"><style:graphic-properties fo:padding="0.5cm"/></style:style>`,
		pages: page(`<draw:custom-shape draw:style-name="g1" svg:x="0cm" svg:y="0cm" svg:width="4cm" svg:height="2cm">` +
			`<text:p>inside</text:p><draw:enhanced-geometry draw:type="rectangle"/></draw:custom-shape>` +
			`<draw:custom-shape svg:x="0cm" svg:y="0cm" svg:width="4cm" svg:height="2cm"><text:p/>` +
			`<draw:enhanced-geometry draw:type="rectangle"/></draw:custom-shape>`),
	}
	d, _ := mustImport(t, f)
	els := d.Slides[0].Elements
	if len(els) != 3 {
		t.Fatalf("elements = %+v", els)
	}
	box := els[1]
	if box.Type != "text" || texts(box.Paragraphs) != "inside" || box.Anchor != "middle" || box.AutoFit != "fixed" ||
		!near(box.Frame.X, 0.5*unitsPerCm, 0.01) || !near(box.Frame.Width, 3*unitsPerCm, 0.02) {
		t.Errorf("shape text = %+v", box)
	}
	if els[2].Type != "shape" {
		t.Errorf("an empty shape is still a shape: %+v", els[2])
	}
}

// TestImportLines draws lines across their frame's diagonal, flipped when
// they run up, with each arrowhead at the end it was drawn at, and leaves
// out a line with no stroke.
func TestImportLines(t *testing.T) {
	f := fixture{
		autoStyles: `<style:style style:name="g1" style:family="graphic"><style:graphic-properties draw:stroke="solid" ` +
			`svg:stroke-color="#0000ff" draw:marker-start="Arrow"/></style:style>`,
		pages: page(`<draw:line draw:style-name="g1" svg:x1="1cm" svg:y1="1cm" svg:x2="3cm" svg:y2="2cm"/>` +
			`<draw:line draw:style-name="g1" svg:x1="3cm" svg:y1="1cm" svg:x2="1cm" svg:y2="2cm"/>` +
			`<draw:connector draw:style-name="g1" svg:x1="1cm" svg:y1="1cm" svg:x2="3cm" svg:y2="1cm"/>` +
			`<draw:line svg:x1="1cm" svg:y1="1cm" svg:x2="3cm" svg:y2="2cm"/>`),
	}
	d, res := mustImport(t, f)
	els := d.Slides[0].Elements
	if len(els) != 3 {
		t.Fatalf("elements = %+v", els)
	}
	down, up := els[0], els[1]
	if down.Flipped || down.StartCap != "arrow" || down.EndCap != "" || down.Stroke.Color != "#0000FF" ||
		!near(down.Frame.Width, 2*unitsPerCm, 0.01) || !near(down.Frame.Height, unitsPerCm, 0.01) {
		t.Errorf("down = %+v", down)
	}
	// Drawn from the right, so its start is the editor's end.
	if !up.Flipped || up.StartCap != "" || up.EndCap != "arrow" {
		t.Errorf("up = %+v", up)
	}
	if !hasWarning(res, "Bent and curved connectors") {
		t.Errorf("warnings = %v", warnings(res))
	}
}

// TestImportRotation turns an element about its center as draw:transform
// says, and keeps it to one turn.
func TestImportRotation(t *testing.T) {
	f := fixture{pages: page(
		`<draw:custom-shape svg:width="4cm" svg:height="2cm" draw:transform="rotate (-1.5707963267949) translate (5cm 1cm)">` +
			`<draw:enhanced-geometry draw:type="rectangle"/></draw:custom-shape>` +
			`<draw:custom-shape svg:width="4cm" svg:height="2cm" draw:transform="rotate (NaN) translate (1cm)">` +
			`<draw:enhanced-geometry draw:type="rectangle"/></draw:custom-shape>`)}
	d, _ := mustImport(t, f)
	el := d.Slides[0].Elements[0]
	// A quarter turn clockwise: the local center (2, 1) goes to (-1, 2),
	// then moves by (5, 1) to (4, 3).
	if el.Frame.Rotation != 90 || !near(el.Frame.X, 2*unitsPerCm, 0.05) || !near(el.Frame.Y, 2*unitsPerCm, 0.05) {
		t.Errorf("frame = %+v", el.Frame)
	}
	if bad := d.Slides[0].Elements[1].Frame; bad.Rotation != 0 || bad.X != 0 {
		t.Errorf("an unreadable transform is ignored: %+v", bad)
	}
}

// TestImportGroups gives a group the box around its children and moves
// them into it, nested.
func TestImportGroups(t *testing.T) {
	rect := func(x, y string) string {
		return `<draw:rect svg:x="` + x + `" svg:y="` + y + `" svg:width="1cm" svg:height="1cm"/>`
	}
	f := fixture{pages: page(`<draw:g>` + rect("2cm", "3cm") + `<draw:g>` + rect("4cm", "5cm") + `</draw:g></draw:g><draw:g/>`)}
	d, _ := mustImport(t, f)
	els := d.Slides[0].Elements
	if len(els) != 1 {
		t.Fatalf("elements = %+v", els)
	}
	g := els[0]
	if g.Type != "group" || !near(g.Frame.X, 2*unitsPerCm, 0.01) || !near(g.Frame.Width, 3*unitsPerCm, 0.02) {
		t.Fatalf("group = %+v", g.Frame)
	}
	if c := g.Children[0].Frame; c.X != 0 || c.Y != 0 {
		t.Errorf("first child = %+v", c)
	}
	inner := g.Children[1]
	if inner.Type != "group" || !near(inner.Frame.X, 2*unitsPerCm, 0.02) || inner.Children[0].Frame.X != 0 {
		t.Errorf("inner group = %+v", inner)
	}
}

// TestImportMasterPages draws a master's own shapes, not its placeholders,
// behind each slide under fresh ids, shows its background where a slide
// sets none, and hides both where the slide says so.
func TestImportMasterPages(t *testing.T) {
	f := fixture{
		masterAuto: `<style:style style:name="Mdp" style:family="drawing-page"><style:drawing-page-properties draw:fill="solid" draw:fill-color="#00aa00"/></style:style>` +
			`<style:style style:name="Mgr" style:family="graphic"><style:graphic-properties draw:fill="solid" draw:fill-color="#aa0000"/></style:style>`,
		masters: `<style:master-page style:name="M" draw:style-name="Mdp">` +
			`<draw:rect draw:style-name="Mgr" svg:x="0cm" svg:y="0cm" svg:width="1cm" svg:height="1cm"/>` +
			textFrame(`presentation:class="title"`, `<text:p>Click to add title</text:p>`) +
			textFrame(`presentation:class="page-number"`, `<text:p><text:page-number>1</text:page-number></text:p>`) +
			`</style:master-page>`,
		autoStyles: `<style:style style:name="own" style:family="drawing-page"><style:drawing-page-properties draw:fill="solid" draw:fill-color="#0000ff"/></style:style>` +
			`<style:style style:name="bare" style:family="drawing-page"><style:drawing-page-properties presentation:background-visible="false" presentation:background-objects-visible="false"/></style:style>`,
		pages: page(textFrame(`presentation:class="title" presentation:placeholder="true"`, "")) + page("") +
			`<draw:page draw:style-name="own" draw:master-page-name="M"/>` + `<draw:page draw:style-name="bare" draw:master-page-name="M"/>`,
	}
	d, _ := mustImport(t, f)
	if len(d.Slides) != 4 {
		t.Fatalf("slides = %d", len(d.Slides))
	}
	ids := map[string]bool{}
	for i, s := range d.Slides[:3] {
		if len(s.Elements) != 1 || s.Elements[0].Fill != "#AA0000" {
			t.Fatalf("slide %d elements = %+v", i+1, s.Elements)
		}
		if ids[s.Elements[0].ID] {
			t.Errorf("slide %d reuses id %s", i+1, s.Elements[0].ID)
		}
		ids[s.Elements[0].ID] = true
	}
	if bg := d.Slides[0].Background; bg == nil || bg.Color != "#00AA00" {
		t.Errorf("master background = %+v", bg)
	}
	if bg := d.Slides[2].Background; bg == nil || bg.Color != "#0000FF" {
		t.Errorf("own background = %+v", bg)
	}
	if s := d.Slides[3]; len(s.Elements) != 0 || s.Background != nil {
		t.Errorf("hidden master = %+v", s)
	}
}

// TestImportTransitions maps the effects the editor has, and warns for the
// rest.
func TestImportTransitions(t *testing.T) {
	style := func(name, props string) string {
		return `<style:style style:name="` + name + `" style:family="drawing-page"><style:drawing-page-properties ` + props + `/></style:style>`
	}
	f := fixture{
		autoStyles: style("fade", `smil:type="fade" smil:subtype="crossfade" presentation:transition-speed="slow"`) +
			style("push", `smil:type="pushWipe" smil:subtype="fromTop"`) +
			style("wipe", `smil:type="barWipe" smil:subtype="leftToRight" smil:direction="reverse"`) +
			style("legacy", `presentation:transition-style="fade-from-left"`) +
			style("cube", `smil:type="miscShapeWipe" smil:subtype="cornersIn"`),
	}
	for _, s := range []string{"fade", "push", "wipe", "legacy", "cube"} {
		f.pages += `<draw:page draw:style-name="` + s + `"/>`
	}
	d, res := mustImport(t, f)
	var got []transition
	for _, s := range d.Slides {
		if s.Transition == nil {
			got = append(got, transition{})
			continue
		}
		got = append(got, *s.Transition)
	}
	want := []transition{{"fade", "", 1000}, {"push", "down", 0}, {"wipe", "", 0}, {"fade", "", 0}, {}}
	if !slices.Equal(got, want) {
		t.Errorf("transitions = %v, want %v", got, want)
	}
	if len(res.Warnings) != 1 || res.Warnings[0].Slide != 5 || !strings.Contains(res.Warnings[0].Message, "transition") {
		t.Errorf("warnings = %+v", res.Warnings)
	}
}

// TestImportWarnsForWhatHasNoModel skips what the editor cannot draw, once
// per slide, with a warning on that slide.
func TestImportWarnsForWhatHasNoModel(t *testing.T) {
	f := fixture{
		manifest: `<manifest:file-entry manifest:full-path="Object 1/" manifest:media-type="application/vnd.oasis.opendocument.chart"/>`,
		pages: page(`<draw:frame svg:width="1cm" svg:height="1cm"><draw:object xlink:href="./Object 1"/></draw:frame>`+
			`<draw:frame svg:width="1cm" svg:height="1cm"><draw:object xlink:href="./Object 2"/></draw:frame>`+
			`<draw:frame svg:width="1cm" svg:height="1cm"><draw:plugin xlink:href="movie.mp4"/></draw:frame>`+
			`<dr3d:scene/><draw:control/><draw:measure/><draw:caption/>`+
			`<anim:par presentation:node-type="timing-root"><anim:seq presentation:node-type="main-sequence">`+
			`<anim:par><anim:set smil:attributeName="visibility"/></anim:par></anim:seq></anim:par>`) +
			page(`<anim:par presentation:node-type="timing-root"><anim:seq presentation:node-type="main-sequence"/></anim:par>`),
	}
	d, res := mustImport(t, f)
	if len(d.Slides[0].Elements) != 0 || len(d.Slides[1].Elements) != 0 {
		t.Errorf("elements = %+v", d.Slides)
	}
	want := []string{
		"Charts are not imported.", "Embedded objects are not imported.", "Video and audio are not imported.",
		"3D objects are not imported.", "Form controls are not imported.",
		"Measurement lines and callouts are not imported.", "Animations are not imported.",
	}
	if got := warnings(res); !slices.Equal(got, want) {
		t.Errorf("warnings = %v, want %v", got, want)
	}
	for _, w := range res.Warnings {
		if w.Slide != 1 {
			t.Errorf("warning on slide %d: %s", w.Slide, w.Message)
		}
	}
}

// TestImportPictures stores each picture once however many slides show
// it, takes a frame's first picture the editor can show, fills a slide
// background from a fill image, and never fetches a linked picture.
func TestImportPictures(t *testing.T) {
	img := func(href string) string {
		return `<draw:frame svg:width="2cm" svg:height="1cm"><draw:image xlink:href="` + href + `"/></draw:frame>`
	}
	f := fixture{
		styles:     `<draw:fill-image draw:name="Bg" xlink:href="Pictures/bg.png"/>`,
		autoStyles: `<style:style style:name="dp" style:family="drawing-page"><style:drawing-page-properties draw:fill="bitmap" draw:fill-image-name="Bg"/></style:style>`,
		pages: page(img("Pictures/a.png")+
			`<draw:frame svg:width="2cm" svg:height="1cm"><draw:image xlink:href="Pictures/v.svg"/><draw:image xlink:href="./Pictures/a.png"/></draw:frame>`) +
			`<draw:page draw:style-name="dp">` + img("Pictures/a.png") + img("https://example.com/x.png") +
			img("../outside.png") + img("Pictures/missing.png") + img("Pictures/v.svg") +
			`<draw:frame svg:width="1cm" svg:height="1cm"><draw:image><office:binary-data>iVBORw0K</office:binary-data></draw:image></draw:frame>` +
			`</draw:page>`,
		extra: []file{
			{"Pictures/a.png", pngBytes}, {"Pictures/bg.png", pngBytes},
			{"Pictures/v.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`)},
		},
	}
	store := &storeAll{}
	d, res, err := importWith(t, f.build(t), store.store)
	if err != nil {
		t.Fatal(err)
	}
	if store.calls != 2 || res.Pictures != 2 || !slices.Equal(store.names, []string{"a.png", "bg.png"}) {
		t.Errorf("stored %v, result %+v", store.names, res)
	}
	s1, s2 := d.Slides[0], d.Slides[1]
	if len(s1.Elements) != 2 || s1.Elements[0].Source != "media/a.png" || s1.Elements[1].Source != "media/a.png" {
		t.Errorf("slide 1 = %+v", s1.Elements)
	}
	if len(s2.Elements) != 1 || s2.Background == nil || s2.Background.Image != "media/bg.png" {
		t.Errorf("slide 2 = %+v %+v", s2.Elements, s2.Background)
	}
	for _, w := range []string{
		"Linked pictures are not imported.", "Pictures missing from the file were left out.",
		"Pictures in a format the editor cannot show (SVG) were left out.",
		"Pictures held inside the slide's XML were left out.",
	} {
		if !slices.Contains(warnings(res), w) {
			t.Errorf("warnings %v lack %q", warnings(res), w)
		}
	}
}

// TestImportWithoutStoreMedia skips every picture, with a warning.
func TestImportWithoutStoreMedia(t *testing.T) {
	f := fixture{
		pages: page(`<draw:frame svg:width="2cm" svg:height="1cm"><draw:image xlink:href="Pictures/a.png"/></draw:frame>`),
		extra: []file{{"Pictures/a.png", pngBytes}},
	}
	d, res, err := importWith(t, f.build(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Slides[0].Elements) != 0 || !slices.Equal(warnings(res), []string{"Pictures are not imported here."}) {
		t.Errorf("slide %+v, warnings %v", d.Slides[0], warnings(res))
	}
}

// TestImportViewModeStoreMedia lets StoreMedia return without reading the
// picture, as a viewer that serves it from the package later does.
func TestImportViewModeStoreMedia(t *testing.T) {
	f := fixture{
		pages: page(`<draw:frame svg:width="2cm" svg:height="1cm"><draw:image xlink:href="Pictures/a.png"/></draw:frame>`),
		extra: []file{{"Pictures/a.png", pngBytes}},
	}
	d, _, err := importWith(t, f.build(t), func(name string, _ io.Reader) (string, error) { return "part:" + name, nil })
	if err != nil {
		t.Fatal(err)
	}
	if src := d.Slides[0].Elements[0].Source; src != "part:a.png" {
		t.Errorf("source = %q", src)
	}
}

// TestImportStoreMediaFailure fails the import when StoreMedia fails for a
// reason of its own.
func TestImportStoreMediaFailure(t *testing.T) {
	f := fixture{
		pages: page(`<draw:frame svg:width="2cm" svg:height="1cm"><draw:image xlink:href="Pictures/a.png"/></draw:frame>`),
		extra: []file{{"Pictures/a.png", pngBytes}},
	}
	boom := errors.New("disk full")
	_, _, err := importWith(t, f.build(t), func(string, io.Reader) (string, error) { return "", boom })
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
}

// TestImportTables reads a table's columns, rows, spans, covered cells,
// repeats, fills, borders and anchors.
func TestImportTables(t *testing.T) {
	f := fixture{
		autoStyles: `<style:style style:name="co" style:family="table-column"><style:table-column-properties style:column-width="2cm"/></style:style>` +
			`<style:style style:name="ro" style:family="table-row"><style:table-row-properties style:row-height="1cm"/></style:style>` +
			`<style:style style:name="ce" style:family="table-cell"><style:graphic-properties draw:fill="solid" draw:fill-color="#ffee00" ` +
			`draw:textarea-vertical-align="middle"/><style:paragraph-properties fo:border="0.1cm solid #ff0000" fo:border-top="none"/></style:style>` +
			`<style:style style:name="ce2" style:family="table-cell"><style:table-cell-properties fo:background-color="#00ffee" fo:border-left="1pt dotted #0000ff"/></style:style>`,
		pages: page(`<draw:frame svg:x="1cm" svg:y="1cm" svg:width="8cm" svg:height="4cm">` +
			`<table:table table:use-first-row-styles="true" table:use-banding-rows-styles="true">` +
			`<table:table-column table:style-name="co" table:number-columns-repeated="2"/><table:table-column/>` +
			`<table:table-row table:style-name="ro"><table:table-cell table:style-name="ce" table:number-columns-spanned="2"><text:p>wide</text:p></table:table-cell>` +
			`<table:covered-table-cell/><table:table-cell table:style-name="ce2"><text:p/></table:table-cell></table:table-row>` +
			`<table:table-rows><table:table-row table:number-rows-repeated="2"><table:table-cell table:number-columns-repeated="3"><text:p>x</text:p></table:table-cell></table:table-row></table:table-rows>` +
			`</table:table></draw:frame>`),
	}
	d, _ := mustImport(t, f)
	tbl := d.Slides[0].Elements[0]
	if tbl.Type != "table" || !tbl.HeaderRow || !tbl.BandedRows || len(tbl.Columns) != 3 || len(tbl.Rows) != 3 {
		t.Fatalf("table = %+v", tbl)
	}
	// The third column takes what the frame has left: 8 - 2 - 2 cm.
	if !near(tbl.Columns[0], 2*unitsPerCm, 0.01) || !near(tbl.Columns[2], 4*unitsPerCm, 0.02) {
		t.Errorf("columns = %v", tbl.Columns)
	}
	if !near(tbl.Rows[0], unitsPerCm, 0.01) || !near(tbl.Rows[1], 1.5*unitsPerCm, 0.02) {
		t.Errorf("rows = %v", tbl.Rows)
	}
	if !near(tbl.Frame.Width, 8*unitsPerCm, 0.05) || !near(tbl.Frame.X, unitsPerCm, 0.01) {
		t.Errorf("frame = %+v", tbl.Frame)
	}
	wide := tbl.Cells[0][0]
	if wide.ColSpan != 2 || texts(wide.Paragraphs) != "wide" || wide.Fill != "#FFEE00" || wide.Anchor != "middle" ||
		wide.Borders == nil || wide.Borders.Top != nil || wide.Borders.Left == nil || *wide.Borders.Left != (stroke{"#FF0000", 6.86, ""}) {
		t.Errorf("wide cell = %+v", wide)
	}
	plain := tbl.Cells[0][2]
	if plain.Paragraphs != nil || plain.Fill != "#00FFEE" || plain.Borders == nil || plain.Borders.Left.Dash != "dot" {
		t.Errorf("plain cell = %+v", plain)
	}
	if texts(tbl.Cells[2][2].Paragraphs) != "x" {
		t.Errorf("repeated cells = %+v", tbl.Cells[2])
	}
}

// TestImportOversizedTable skips a table past the editor's limits, with a
// warning, however large a repeat count it declares.
func TestImportOversizedTable(t *testing.T) {
	for _, body := range []string{
		`<table:table-column table:number-columns-repeated="1000000000"/>`,
		`<table:table-column/><table:table-row table:number-rows-repeated="2147483647"><table:table-cell/></table:table-row>`,
		`<table:table-row><table:table-cell table:number-columns-repeated="99999999999999999999"/></table:table-row>`,
		strings.Repeat(`<table:table-row><table:table-cell/></table:table-row>`, 501),
	} {
		f := fixture{pages: page(`<draw:frame svg:width="8cm" svg:height="4cm"><table:table>` + body + `</table:table></draw:frame>`)}
		d, res := mustImport(t, f)
		if len(d.Slides[0].Elements) != 0 || !hasWarning(res, "Tables of more than") {
			t.Errorf("%.60s: elements %+v, warnings %v", body, d.Slides[0].Elements, warnings(res))
		}
	}
}
