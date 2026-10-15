package odputil_test

// cspell:ignore IEND smil xlink

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/odputil"
)

// namespaces declares every prefix the fixtures use.
const namespaces = `xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" ` +
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
	`xmlns:anim="urn:oasis:names:tc:opendocument:xmlns:animation:1.0" ` +
	`xmlns:smil="urn:oasis:names:tc:opendocument:xmlns:smil-compatible:1.0" ` +
	`xmlns:dr3d="urn:oasis:names:tc:opendocument:xmlns:dr3d:1.0" ` +
	`xmlns:form="urn:oasis:names:tc:opendocument:xmlns:form:1.0" ` +
	`xmlns:manifest="urn:oasis:names:tc:opendocument:xmlns:manifest:1.0"`

// odpMime is an .odp's mimetype.
const odpMime = "application/vnd.oasis.opendocument.presentation"

// pngBytes is a 1×1 PNG.
var pngBytes = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00\x90wS\xde" +
	"\x00\x00\x00\x0cIDATx\x9cc\xf8\xcf\xc0\x00\x00\x03\x01\x01\x00\xc9\xfe\x92\xef\x00\x00\x00\x00IEND\xaeB`\x82")

// file is one entry of a fixture package.
type file struct {
	name string
	body []byte
}

// fixture is an .odp to build: its parts' contents, each wrapped in its
// root element. An empty styles or meta leaves the part out.
type fixture struct {
	mime       string
	noMime     bool
	contentRaw string
	autoStyles string
	pages      string
	styles     string
	masterAuto string
	masters    string
	meta       string
	manifest   string
	extra      []file
}

// content is the fixture's content.xml.
func (f fixture) content() string {
	if f.contentRaw != "" {
		return f.contentRaw
	}
	return `<?xml version="1.0" encoding="UTF-8"?><office:document-content ` + namespaces + `>` +
		`<office:automatic-styles>` + f.autoStyles + `</office:automatic-styles>` +
		`<office:body><office:presentation>` + f.pages + `</office:presentation></office:body></office:document-content>`
}

// build writes the fixture as a zip, its mimetype first and stored, as
// ODF packages it.
func (f fixture) build(t testing.TB) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	add := func(name string, body []byte, method uint16) {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: method})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if !f.noMime {
		mime := f.mime
		if mime == "" {
			mime = odpMime
		}
		add("mimetype", []byte(mime), zip.Store)
	}
	add("content.xml", []byte(f.content()), zip.Deflate)
	if f.styles != "" || f.masterAuto != "" || f.masters != "" {
		add("styles.xml", []byte(`<?xml version="1.0" encoding="UTF-8"?><office:document-styles `+namespaces+`>`+
			`<office:styles>`+f.styles+`</office:styles>`+
			`<office:automatic-styles>`+f.masterAuto+`</office:automatic-styles>`+
			`<office:master-styles>`+f.masters+`</office:master-styles></office:document-styles>`), zip.Deflate)
	}
	if f.meta != "" {
		add("meta.xml", []byte(`<?xml version="1.0" encoding="UTF-8"?><office:document-meta `+namespaces+
			`><office:meta>`+f.meta+`</office:meta></office:document-meta>`), zip.Deflate)
	}
	if f.manifest != "" {
		add("META-INF/manifest.xml", []byte(`<?xml version="1.0" encoding="UTF-8"?><manifest:manifest `+namespaces+
			`>`+f.manifest+`</manifest:manifest>`), zip.Deflate)
	}
	for _, e := range f.extra {
		add(e.name, e.body, zip.Deflate)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// deck is a .qslide as the tests read it back.
type deck struct {
	SchemaVersion int     `json:"schemaVersion"`
	Title         string  `json:"title"`
	Size          size    `json:"size"`
	Slides        []slide `json:"slides"`
}

type size struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type slide struct {
	ID         string      `json:"id"`
	Background *background `json:"background"`
	Elements   []element   `json:"elements"`
	Notes      string      `json:"notes"`
	Transition *transition `json:"transition"`
}

type background struct {
	Color string `json:"color"`
	Image string `json:"image"`
}

type transition struct {
	Kind      string `json:"kind"`
	Direction string `json:"direction"`
	Duration  int    `json:"duration"`
}

type frame struct {
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Rotation float64 `json:"rotation"`
}

type element struct {
	ID           string      `json:"id"`
	Type         string      `json:"type"`
	Frame        frame       `json:"frame"`
	Paragraphs   []paragraph `json:"paragraphs"`
	Anchor       string      `json:"anchor"`
	AutoFit      string      `json:"autoFit"`
	Kind         string      `json:"kind"`
	Fill         string      `json:"fill"`
	Stroke       *stroke     `json:"stroke"`
	CornerRadius *float64    `json:"cornerRadius"`
	Opacity      *float64    `json:"opacity"`
	Source       string      `json:"source"`
	AltText      string      `json:"altText"`
	Fit          string      `json:"fit"`
	Flipped      bool        `json:"flipped"`
	StartCap     string      `json:"startCap"`
	EndCap       string      `json:"endCap"`
	Children     []element   `json:"children"`
	Columns      []float64   `json:"columns"`
	Rows         []float64   `json:"rows"`
	HeaderRow    bool        `json:"headerRow"`
	BandedRows   bool        `json:"bandedRows"`
	Cells        [][]cell    `json:"cells"`
}

type paragraph struct {
	Runs        []run    `json:"runs"`
	Align       string   `json:"align"`
	LineSpacing *float64 `json:"lineSpacing"`
	List        string   `json:"list"`
}

type run struct {
	Text          string   `json:"text"`
	Bold          bool     `json:"bold"`
	Italic        bool     `json:"italic"`
	Underline     bool     `json:"underline"`
	Strikethrough bool     `json:"strikethrough"`
	FontSize      *float64 `json:"fontSize"`
	FontFamily    *string  `json:"fontFamily"`
	Color         string   `json:"color"`
}

type stroke struct {
	Color string  `json:"color"`
	Width float64 `json:"width"`
	Dash  string  `json:"dash"`
}

type cell struct {
	Paragraphs []paragraph `json:"paragraphs"`
	Fill       string      `json:"fill"`
	Borders    *struct {
		Top    *stroke `json:"top"`
		Right  *stroke `json:"right"`
		Bottom *stroke `json:"bottom"`
		Left   *stroke `json:"left"`
	} `json:"borders"`
	Anchor  string `json:"anchor"`
	RowSpan int    `json:"rowSpan"`
	ColSpan int    `json:"colSpan"`
}

// storeAll is a StoreMedia that reads each picture and names it by its
// stored name, counting the calls.
type storeAll struct {
	calls int
	names []string
}

func (s *storeAll) store(name string, r io.Reader) (string, error) {
	s.calls++
	s.names = append(s.names, name)
	if _, err := io.Copy(io.Discard, r); err != nil {
		return "", err
	}
	return "media/" + name, nil
}

// importBytes imports an .odp held in memory, with every picture stored.
func importBytes(t testing.TB, data []byte) (deck, odputil.ImportOdpResult, error) {
	t.Helper()
	store := &storeAll{}
	return importWith(t, data, store.store)
}

// importWith imports an .odp held in memory with the given StoreMedia.
func importWith(t testing.TB, data []byte, storeMedia odputil.StoreMediaFunc) (deck, odputil.ImportOdpResult, error) {
	t.Helper()
	var out bytes.Buffer
	res, err := odputil.ImportOdp(odputil.ImportOdpParams{
		Source: bytes.NewReader(data), Size: int64(len(data)), Out: &out, StoreMedia: storeMedia, Title: "Fallback",
	})
	if err != nil {
		return deck{}, res, err
	}
	if !json.Valid(out.Bytes()) {
		t.Fatalf("the import wrote invalid JSON:\n%s", out.String())
	}
	var d deck
	if err := json.Unmarshal(out.Bytes(), &d); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return d, res, nil
}

// mustImport imports a fixture and fails the test on an error.
func mustImport(t testing.TB, f fixture) (deck, odputil.ImportOdpResult) {
	t.Helper()
	d, res, err := importBytes(t, f.build(t))
	if err != nil {
		t.Fatalf("ImportOdp: %v", err)
	}
	return d, res
}

// page is one draw:page holding body.
func page(body string) string {
	return `<draw:page draw:name="p" draw:master-page-name="M">` + body + `</draw:page>`
}

// textFrame is a text frame at 0,0 of 10×5 cm holding paragraphs, with
// the given extra attributes.
func textFrame(attrs, paragraphs string) string {
	return `<draw:frame svg:x="0cm" svg:y="0cm" svg:width="10cm" svg:height="5cm" ` + attrs + `><draw:text-box>` +
		paragraphs + `</draw:text-box></draw:frame>`
}

// warnings is every warning message of a result, deck and slides.
func warnings(res odputil.ImportOdpResult) []string {
	var out []string
	for _, w := range res.Warnings {
		out = append(out, w.Message)
	}
	return out
}

// hasWarning reports whether a result carries a warning containing part.
func hasWarning(res odputil.ImportOdpResult, part string) bool {
	for _, w := range res.Warnings {
		if strings.Contains(w.Message, part) {
			return true
		}
	}
	return false
}

// texts is the text of every run of every paragraph, a line per paragraph.
func texts(ps []paragraph) string {
	var lines []string
	for _, p := range ps {
		var b strings.Builder
		for _, r := range p.Runs {
			b.WriteString(r.Text)
		}
		lines = append(lines, b.String())
	}
	return strings.Join(lines, "\n")
}

// near reports whether two lengths agree to within a hundredth of a unit
// per unit of tolerance.
func near(a, b, tolerance float64) bool {
	d := a - b
	return d <= tolerance && d >= -tolerance
}
