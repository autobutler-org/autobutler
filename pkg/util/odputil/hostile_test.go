package odputil_test

// cspell:ignore rects Infcm smil xlink

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/odputil"
)

// TestImportRefusesWhatIsNotAPresentation answers ErrNotOdp for a file
// that is not an OpenDocument presentation.
func TestImportRefusesWhatIsNotAPresentation(t *testing.T) {
	for name, data := range map[string][]byte{
		"empty":            {},
		"not a zip":        []byte("hello, world"),
		"a text document":  fixture{mime: "application/vnd.oasis.opendocument.text"}.build(t),
		"no content":       zipOf(t, file{"mimetype", []byte(odpMime)}),
		"no presentation":  fixture{contentRaw: `<office:document-content ` + namespaces + `><office:body><office:text/></office:body></office:document-content>`}.build(t),
		"another root":     fixture{contentRaw: `<office:document-styles ` + namespaces + `/>`}.build(t),
		"malformed XML":    fixture{pages: page(`<draw:frame>`)}.build(t),
		"truncated XML":    fixture{contentRaw: `<office:document-content ` + namespaces + `><office:body>`}.build(t),
		"unsafe entry":     zipOf(t, file{"../content.xml", []byte(fixture{}.content())}),
		"undeclared names": fixture{contentRaw: `<office:document-content><office:body/></office:document-content>`}.build(t),
	} {
		var out bytes.Buffer
		_, err := odputil.ImportOdp(odputil.ImportOdpParams{Source: bytes.NewReader(data), Size: int64(len(data)), Out: &out})
		if !errors.Is(err, odputil.ErrNotOdp) {
			t.Errorf("%s: err = %v, want ErrNotOdp", name, err)
		}
	}
}

// TestImportWithoutMimetype reads a package that leaves its mimetype out,
// as some writers do, by its content.
func TestImportWithoutMimetype(t *testing.T) {
	d, _ := mustImport(t, fixture{noMime: true, pages: page("")})
	if len(d.Slides) != 1 {
		t.Fatalf("slides = %d", len(d.Slides))
	}
}

// TestImportRefusesAnEncryptedPresentation answers ErrEncrypted when the
// manifest says the content is encrypted.
func TestImportRefusesAnEncryptedPresentation(t *testing.T) {
	data := fixture{manifest: `<manifest:file-entry manifest:full-path="content.xml" manifest:media-type="text/xml">` +
		`<manifest:encryption-data manifest:checksum="x"/></manifest:file-entry>`}.build(t)
	if _, _, err := importBytes(t, data); !errors.Is(err, odputil.ErrEncrypted) {
		t.Fatalf("err = %v", err)
	}
}

// TestImportRequiresSourceAndOut refuses a call without either.
func TestImportRequiresSourceAndOut(t *testing.T) {
	if _, err := odputil.ImportOdp(odputil.ImportOdpParams{}); err == nil {
		t.Fatal("no error")
	}
}

// TestImportLimits answers ErrTooLarge past each of the import's limits.
func TestImportLimits(t *testing.T) {
	rects := strings.Repeat(`<draw:rect svg:width="1cm" svg:height="1cm"/>`, 100)
	nested := strings.Repeat(`<draw:g>`, odputil.MaxGroupDepth+1) + `<draw:rect/>` + strings.Repeat(`</draw:g>`, odputil.MaxGroupDepth+1)
	deep := strings.Repeat(`<text:span>`, odputil.MaxXMLDepth) + "x" + strings.Repeat(`</text:span>`, odputil.MaxXMLDepth)
	var styles strings.Builder
	for i := range odputil.MaxStyleDefinitions + 1 {
		styles.WriteString(`<style:style style:family="graphic" style:name="s`)
		styles.WriteString(strings.Repeat("x", i%3))
		styles.WriteString(`"/>`)
	}
	spaces := strings.Repeat(`<text:s text:c="1000"/>`, odputil.MaxSlideTextBytes/1000+1)
	for name, f := range map[string]fixture{
		"slides":        {pages: strings.Repeat(`<draw:page/>`, odputil.MaxSlides+1)},
		"master copies": {masters: `<style:master-page style:name="M">` + rects + `</style:master-page>`, pages: strings.Repeat(page(""), odputil.MaxElements/100+1)},
		"group depth":   {pages: page(nested)},
		"XML depth":     {pages: page(textFrame("", "<text:p>"+deep+"</text:p>"))},
		"styles":        {autoStyles: styles.String()},
		"slide text":    {pages: page(textFrame("", "<text:p>"+spaces+"</text:p>"))},
	} {
		_, _, err := importBytes(t, f.build(t))
		if !errors.Is(err, odputil.ErrTooLarge) {
			t.Errorf("%s: err = %v, want ErrTooLarge", name, err)
		}
	}
}

// TestImportStyleCycle ends a style whose parents loop back to it.
func TestImportStyleCycle(t *testing.T) {
	f := fixture{
		styles: `<style:style style:name="A" style:family="graphic" style:parent-style-name="B"/>` +
			`<style:style style:name="B" style:family="graphic" style:parent-style-name="A"/>`,
		pages: page(textFrame(`draw:style-name="A"`, `<text:p>x</text:p>`)),
	}
	d, _ := mustImport(t, f)
	if texts(d.Slides[0].Elements[0].Paragraphs) != "x" {
		t.Fatalf("slide = %+v", d.Slides[0])
	}
}

// TestImportHostileGeometry keeps every length finite and bounded however
// large or strange a file's numbers are.
func TestImportHostileGeometry(t *testing.T) {
	f := fixture{pages: page(
		`<draw:rect svg:x="1e308cm" svg:y="-Infcm" svg:width="NaNcm" svg:height="99999999cm"/>` +
			`<draw:rect svg:width="1cm" svg:height="1cm" draw:transform="scale (100000 100000) translate (1e300cm 0cm) matrix (1 2 3)"/>` +
			`<draw:rect svg:width="1cm" svg:height="1cm" draw:transform="` + strings.Repeat("rotate (1) ", 1000) + `"/>` +
			`<draw:line svg:x1="12q" svg:y1="" svg:x2="1e999cm" svg:y2="5cm" draw:style-name="none"/>`)}
	d, _ := mustImport(t, f)
	for _, el := range d.Slides[0].Elements {
		for _, v := range []float64{el.Frame.X, el.Frame.Y, el.Frame.Width, el.Frame.Height} {
			if v > 1e6 || v < -1e6 {
				t.Errorf("unbounded frame %+v", el.Frame)
			}
		}
	}
}

// TestImportOversizedPicture leaves out a picture past MaxImageBytes with
// a warning, having streamed it rather than held it.
func TestImportOversizedPicture(t *testing.T) {
	if testing.Short() {
		t.Skip("streams 100 MiB")
	}
	big := pictureZip(t, odputil.MaxImageBytes+1)
	var stored int64
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	d, res, err := importWith(t, big, func(name string, r io.Reader) (string, error) {
		n, err := io.Copy(io.Discard, r)
		stored += n
		return name, err
	})
	runtime.ReadMemStats(&after)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Slides[0].Elements) != 0 || !hasWarning(res, "Pictures larger than 100 MiB were left out.") {
		t.Errorf("elements %+v, warnings %v", d.Slides[0].Elements, warnings(res))
	}
	if stored != odputil.MaxImageBytes {
		t.Errorf("StoreMedia read %d bytes", stored)
	}
	if grew := after.TotalAlloc - before.TotalAlloc; grew > 16<<20 {
		t.Errorf("importing a 100 MiB picture allocated %d MiB", grew>>20)
	}
}

// pictureZip is an .odp whose one slide shows a PNG of n bytes, mostly
// zeros, so the zip stays small.
func pictureZip(t testing.TB, n int64) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("content.xml")
	if err != nil {
		t.Fatal(err)
	}
	io.WriteString(w, fixture{pages: page(`<draw:frame svg:width="2cm" svg:height="1cm"><draw:image xlink:href="Pictures/big.png"/></draw:frame>`)}.content())
	if w, err = zw.Create("Pictures/big.png"); err != nil {
		t.Fatal(err)
	}
	w.Write(pngBytes)
	if _, err := io.CopyN(w, zeros{}, n-int64(len(pngBytes))); err != nil {
		t.Fatal(err)
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

// zipOf is a zip of the given files, nothing else.
func zipOf(t testing.TB, files ...file) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range files {
		w, err := zw.Create(f.name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write(f.body)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// FuzzImportOdp feeds the reader packages built around arbitrary
// content.xml, and arbitrary files, and checks that it never panics and
// that whatever it writes without an error is JSON.
func FuzzImportOdp(f *testing.F) {
	if sample, err := os.ReadFile("testdata/sample.odp"); err == nil {
		f.Add(sample, "")
	}
	f.Add([]byte(nil), fixture{pages: page(textFrame("", `<text:p>a<text:span>b</text:span></text:p>`))}.content())
	f.Add([]byte(nil), fixture{pages: page(`<draw:frame><table:table><table:table-row><table:table-cell/></table:table-row></table:table></draw:frame>`)}.content())
	f.Add([]byte(nil), fixture{pages: page(`<draw:g><draw:line svg:x1="1cm"/><draw:custom-shape><draw:enhanced-geometry draw:type="star5"/></draw:custom-shape></draw:g>`)}.content())
	f.Fuzz(func(t *testing.T, raw []byte, content string) {
		data := raw
		if content != "" {
			data = zipOf(t, file{"mimetype", []byte(odpMime)}, file{"content.xml", []byte(content)},
				file{"Pictures/a.png", pngBytes})
		}
		var out bytes.Buffer
		_, err := odputil.ImportOdp(odputil.ImportOdpParams{
			Source: bytes.NewReader(data), Size: int64(len(data)), Out: &out,
			StoreMedia: func(name string, r io.Reader) (string, error) { _, err := io.Copy(io.Discard, r); return name, err },
		})
		if err == nil && !json.Valid(out.Bytes()) {
			t.Fatalf("invalid JSON for %q", content)
		}
	})
}
