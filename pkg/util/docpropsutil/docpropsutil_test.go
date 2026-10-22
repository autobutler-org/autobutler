package docpropsutil_test

// cspell:ignore dcterms fonttbl Msftedit nofchars

import (
	"archive/zip"
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/docpropsutil"
)

func zipOf(t *testing.T, parts map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range parts {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func extract(t *testing.T, name string, data []byte) (docpropsutil.Properties, error) {
	t.Helper()
	res, err := docpropsutil.Extract(docpropsutil.ExtractParams{
		Source: bytes.NewReader(data), Size: int64(len(data)), Name: name,
	})
	return res.Properties, err
}

func TestExtract_OOXML(t *testing.T) {
	data := zipOf(t, map[string]string{
		"docProps/core.xml": `<cp:coreProperties xmlns:cp="c" xmlns:dc="d" xmlns:dcterms="t">` +
			`<dc:title>Quarterly</dc:title><dc:creator>Ada</dc:creator><dc:subject>Money</dc:subject>` +
			`<dcterms:created>2024-03-01T10:00:00Z</dcterms:created><dcterms:modified>2024-03-02T11:30:00Z</dcterms:modified>` +
			`</cp:coreProperties>`,
		"docProps/app.xml": `<Properties xmlns="a" xmlns:vt="v"><Application>Microsoft Office Word</Application>` +
			`<Pages>7</Pages><Words>1200</Words><Characters>6400</Characters><Slides>12</Slides>` +
			`<HeadingPairs><vt:vector size="4" baseType="variant"><vt:variant><vt:lpstr>Worksheets</vt:lpstr></vt:variant>` +
			`<vt:variant><vt:i4>3</vt:i4></vt:variant></vt:vector></HeadingPairs></Properties>`,
	})
	got, err := extract(t, "report.docx", data)
	if err != nil {
		t.Fatal(err)
	}
	want := docpropsutil.Properties{
		Title: "Quarterly", Author: "Ada", Subject: "Money",
		Created: "2024-03-01T10:00:00Z", Modified: "2024-03-02T11:30:00Z",
		Application: "Microsoft Office Word", Pages: 7, Words: 1200, Characters: 6400, Slides: 12, Sheets: 3,
		CountsSource: "saved",
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestExtract_OOXMLWithoutMetadataIsEmpty(t *testing.T) {
	for _, name := range []string{"a.docx", "a.pptx", "a.xlsx"} {
		got, err := extract(t, name, zipOf(t, map[string]string{"word/document.xml": "<x/>"}))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got != (docpropsutil.Properties{CountsSource: "saved"}) {
			t.Errorf("%s: got %+v, want only countsSource", name, got)
		}
	}
}

func TestExtract_OOXMLOddValues(t *testing.T) {
	data := zipOf(t, map[string]string{
		"docProps/core.xml": `<coreProperties><created>last tuesday</created><title>  padded  </title></coreProperties>`,
		"docProps/app.xml":  `<Properties><Pages>-4</Pages><Words>many</Words></Properties>`,
	})
	got, err := extract(t, "odd.docx", data)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "padded" || got.Created != "last tuesday" || got.Pages != 0 || got.Words != 0 {
		t.Errorf("got %+v", got)
	}
}

func TestExtract_ODF(t *testing.T) {
	meta := func(stat string) string {
		return `<office:document-meta xmlns:office="o" xmlns:meta="m" xmlns:dc="d"><office:meta>` +
			`<dc:title>Notes</dc:title><dc:subject>S</dc:subject><meta:initial-creator>Grace</meta:initial-creator>` +
			`<dc:creator>Someone Else</dc:creator><meta:creation-date>2023-01-05T08:00:00.5</meta:creation-date>` +
			`<dc:date>2023-02-06T09:00:00</dc:date><meta:generator>LibreOffice/7.6</meta:generator>` +
			`<meta:document-statistic ` + stat + `/></office:meta></office:document-meta>`
	}
	const stat = `meta:page-count="4" meta:word-count="300" meta:character-count="1500" meta:table-count="2"`
	for name, want := range map[string]docpropsutil.Properties{
		"a.odt": {Pages: 4},
		"a.odp": {Slides: 4},
		"a.ods": {Sheets: 2},
	} {
		got, err := extract(t, name, zipOf(t, map[string]string{"meta.xml": meta(stat)}))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		want.Title, want.Subject, want.Author = "Notes", "S", "Grace"
		want.Created, want.Modified = "2023-01-05T08:00:00Z", "2023-02-06T09:00:00Z"
		want.Application, want.Words, want.Characters, want.CountsSource = "LibreOffice/7.6", 300, 1500, "saved"
		if got != want {
			t.Errorf("%s: got %+v, want %+v", name, got, want)
		}
	}
}

func TestExtract_ODFFallsBackToLastModifierAndToEmpty(t *testing.T) {
	got, err := extract(t, "a.odt", zipOf(t, map[string]string{
		"meta.xml": `<document-meta><meta><creator>Only Me</creator></meta></document-meta>`,
	}))
	if err != nil || got.Author != "Only Me" {
		t.Errorf("got %+v, %v", got, err)
	}
	got, err = extract(t, "a.odt", zipOf(t, map[string]string{"content.xml": "<x/>"}))
	if err != nil || got != (docpropsutil.Properties{CountsSource: "saved"}) {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestExtract_RTF(t *testing.T) {
	rtf := `{\rtf1\ansi\deff0{\fonttbl{\f0 Arial;}}{\*\generator Msftedit 5.41;}` +
		`{\info{\title Ren\'e9 \{plan\}}{\author Ada}{\subject Q1}{\*\company Acme}` +
		`{\creatim\yr2022\mo4\dy9\hr13\min5}{\revtim\yr2022\mo5\dy1\hr0\min0}` +
		`\nofpages3\nofwords250\nofchars1300}\pard The body mentions \nofpages99.\par}`
	got, err := extract(t, "a.rtf", []byte(rtf))
	if err != nil {
		t.Fatal(err)
	}
	want := docpropsutil.Properties{
		Title: "Ren\u00e9 {plan}", Author: "Ada", Subject: "Q1",
		Created: "2022-04-09T13:05:00Z", Modified: "2022-05-01T00:00:00Z",
		Pages: 3, Words: 250, Characters: 1300, Application: "Msftedit 5.41", CountsSource: "saved",
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestExtract_RTFWithoutInfoIsEmpty(t *testing.T) {
	got, err := extract(t, "a.rtf", []byte(`{\rtf1 hello \nofpages9}`))
	if err != nil || got != (docpropsutil.Properties{CountsSource: "saved"}) {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestExtract_RTFScansOnlyItsHeader(t *testing.T) {
	// An \info group placed after the cap is never reached, and the scan
	// does not read the filler into memory to look for it.
	pad := strings.Repeat("x", docpropsutil.MaxRTFHeaderBytes+10)
	got, err := extract(t, "a.rtf", []byte(`{\rtf1 `+pad+`{\info{\title Late}}}`))
	if err != nil || got.Title != "" {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestExtract_RTFDeepNestingIsMalformed(t *testing.T) {
	_, err := extract(t, "a.rtf", []byte(`{\rtf1`+strings.Repeat("{", 1000)))
	if !errors.Is(err, docpropsutil.ErrMalformed) {
		t.Errorf("err = %v, want ErrMalformed", err)
	}
}

func TestExtract_Refusals(t *testing.T) {
	for name, tc := range map[string]struct {
		file string
		data []byte
		want error
	}{
		"not a zip":        {"a.docx", []byte("plain text"), docpropsutil.ErrMalformed},
		"not an rtf":       {"a.rtf", []byte("plain text"), docpropsutil.ErrMalformed},
		"unknown format":   {"a.txt", []byte("x"), docpropsutil.ErrUnsupportedFormat},
		"broken xml":       {"a.docx", zipOf(t, map[string]string{"docProps/core.xml": "<a><b></a>"}), docpropsutil.ErrMalformed},
		"broken odf xml":   {"a.odt", zipOf(t, map[string]string{"meta.xml": "not xml <"}), docpropsutil.ErrMalformed},
		"oversize entry":   {"a.docx", zipOf(t, map[string]string{"docProps/core.xml": "<a>" + strings.Repeat("x", docpropsutil.MaxMetadataBytes+1) + "</a>"}), docpropsutil.ErrTooLarge},
		"oversize meta":    {"a.odt", zipOf(t, map[string]string{"meta.xml": "<a>" + strings.Repeat("x", docpropsutil.MaxMetadataBytes+1) + "</a>"}), docpropsutil.ErrTooLarge},
		"too many element": {"a.docx", zipOf(t, map[string]string{"docProps/app.xml": "<a>" + strings.Repeat("<b/>", 100_001) + "</a>"}), docpropsutil.ErrTooLarge},
	} {
		if _, err := extract(t, tc.file, tc.data); !errors.Is(err, tc.want) {
			t.Errorf("%s: err = %v, want %v", name, err, tc.want)
		}
	}
}
