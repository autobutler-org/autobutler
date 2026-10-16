package odsutil

// cspell:ignore calcext documentfoundation xmlns

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"runtime"
	"strings"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/officezip"
)

// entry is one part of a test package.
type entry struct {
	name, body string
}

func packageBytes(content, mime, manifestExtra string, extra ...entry) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	parts := []entry{
		{name: "mimetype", body: mime},
		{name: "META-INF/manifest.xml", body: manifestXML(mime, manifestExtra)},
		{name: "content.xml", body: content},
	}
	parts = append(parts, extra...)
	for _, p := range parts {
		w, err := zw.Create(p.name)
		if err != nil {
			return nil, err
		}
		if _, err := io.WriteString(w, p.body); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func manifestXML(mime, extra string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>` +
		`<manifest:manifest xmlns:manifest="urn:oasis:names:tc:opendocument:xmlns:manifest:1.0" manifest:version="1.3">` +
		`<manifest:file-entry manifest:full-path="/" manifest:media-type="` + mime + `"/>` +
		`<manifest:file-entry manifest:full-path="content.xml" manifest:media-type="text/xml"/>` +
		extra +
		`</manifest:manifest>`
}

func contentXML(inner string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>` +
		`<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" ` +
		`xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0" ` +
		`xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0" ` +
		`xmlns:calcext="urn:org:documentfoundation:names:experimental:calc:xmlns:calcext:1.0">` +
		`<office:body><office:spreadsheet>` + inner + `</office:spreadsheet></office:body></office:document-content>`
}

func sheet(name, rows string) string {
	return `<table:table table:name="` + name + `">` + rows + `</table:table>`
}

func row(cells string) string {
	return `<table:table-row>` + cells + `</table:table-row>`
}

func strCell(text string) string {
	return `<table:table-cell office:value-type="string"><text:p>` + text + `</text:p></table:table-cell>`
}

func floatCell(value, shown string) string {
	return `<table:table-cell office:value-type="float" office:value="` + value + `"><text:p>` + shown + `</text:p></table:table-cell>`
}

// qsheetDoc is the envelope decoded back for assertions.
type qsheetDoc struct {
	Tabs []struct {
		Name string `json:"name"`
		Data struct {
			Rows [][]string `json:"rows"`
		} `json:"data"`
	} `json:"tabs"`
}

func convertPackage(t *testing.T, data []byte) (qsheetDoc, ConvertToQsheetResult, string) {
	t.Helper()
	var out bytes.Buffer
	result, err := ConvertToQsheet(ConvertToQsheetParams{Source: bytes.NewReader(data), Size: int64(len(data)), Out: &out})
	if err != nil {
		t.Fatalf("ConvertToQsheet: %v", err)
	}
	var doc qsheetDoc
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatalf("envelope is not JSON: %v\n%s", err, out.String())
	}
	return doc, result, out.String()
}

func convertSheet(t *testing.T, inner string, extra ...entry) (qsheetDoc, ConvertToQsheetResult) {
	t.Helper()
	data, err := packageBytes(contentXML(inner), mimeSpreadsheet, "", extra...)
	if err != nil {
		t.Fatal(err)
	}
	doc, result, _ := convertPackage(t, data)
	return doc, result
}

func assertRows(t *testing.T, got, want [][]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("rows = %q, want %q", got, want)
	}
	for i := range want {
		if len(got[i]) != len(want[i]) {
			t.Fatalf("row %d = %q, want %q", i, got[i], want[i])
		}
		for j := range want[i] {
			if got[i][j] != want[i][j] {
				t.Errorf("cell %d,%d = %q, want %q", i, j, got[i][j], want[i][j])
			}
		}
	}
}

func wantErr(t *testing.T, data []byte, sentinel error) {
	t.Helper()
	var out bytes.Buffer
	_, err := ConvertToQsheet(ConvertToQsheetParams{Source: bytes.NewReader(data), Size: int64(len(data)), Out: &out})
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
	if out.Len() != 0 {
		t.Fatalf("wrote %q before failing", out.String())
	}
}

func wantContentErr(t *testing.T, inner string, sentinel error) {
	t.Helper()
	data, err := packageBytes(contentXML(inner), mimeSpreadsheet, "")
	if err != nil {
		t.Fatal(err)
	}
	wantErr(t, data, sentinel)
}

func TestConvertToQsheetCellTypes(t *testing.T) {
	doc, result := convertSheet(t, sheet("Budget",
		row(strCell("Item")+strCell("Cost"))+
			row(strCell("Coffee")+floatCell("4.5", "$4.50"))+
			row(strCell("Total")+`<table:table-cell table:formula="of:=SUM([.B2])" office:value-type="float" office:value="9"><text:p>9</text:p></table:table-cell>`)+
			row(`<table:table-cell office:value-type="boolean" office:boolean-value="true"><text:p>yes</text:p></table:table-cell>`+
				`<table:table-cell office:value-type="boolean" office:boolean-value="false"/>`)+
			row(`<table:table-cell table:formula="of:=1/0" office:value-type="string" office:string-value="#DIV/0!" calcext:value-type="error"><text:p>#DIV/0!</text:p></table:table-cell>`+
				`<table:table-cell office:value-type="percentage" office:value="0.25"><text:p>25%</text:p></table:table-cell>`)+
			row(`<table:table-cell office:value-type="currency" office:currency="USD" office:value="12.5"><text:p>$12.50</text:p></table:table-cell>`+
				`<table:table-cell office:value-type="date" office:date-value="2023-03-15"><text:p>03/15/2023</text:p></table:table-cell>`)+
			row(`<table:table-cell office:value-type="date" office:date-value="2023-03-15T12:00:00"><text:p>noon</text:p></table:table-cell>`+
				`<table:table-cell office:value-type="time" office:time-value="PT1H30M00S"><text:p>1:30</text:p></table:table-cell>`)+
			row(`<table:table-cell office:value-type="time" office:time-value="PT26H05M09S"/>`+
				`<table:table-cell office:value-type="time" office:time-value="-PT1H"/>`)+
			row(`<table:table-cell office:value-type="date" office:date-value="2023-03-15T12:00:00+02:00"/>`+
				`<table:table-cell office:value-type="date" office:date-value="nope"><text:p>March</text:p></table:table-cell>`)+
			row(`<table:table-cell office:value-type="string" office:string-value="via-attr"/>`+
				`<table:table-cell table:formula="of:=1+1"/>`),
	))

	if doc.Tabs[0].Name != "Budget" {
		t.Errorf("name = %q", doc.Tabs[0].Name)
	}
	assertRows(t, doc.Tabs[0].Data.Rows, [][]string{
		{"Item", "Cost"},
		{"Coffee", "4.5"},
		{"Total", "9"},
		{"TRUE", "FALSE"},
		{"#DIV/0!", "0.25"},
		{"12.5", "2023-03-15"},
		{"2023-03-15 12:00:00", "01:30:00"},
		{"26:05:09", "-01:00:00"},
		{"2023-03-15 12:00:00", "March"},
		{"via-attr", ""},
	})
	if result.Tabs != 1 || result.Rows != 10 || result.Cells != 19 {
		t.Fatalf("result = %+v", result)
	}
}

func TestConvertToQsheetText(t *testing.T) {
	doc, _ := convertSheet(t, sheet("T", row(
		`<table:table-cell office:value-type="string"><text:p>Rich<text:span> Text</text:span></text:p><text:p>Line 2</text:p></table:table-cell>`+
			`<table:table-cell office:value-type="string"><text:p>A<text:s text:c="3"/>B<text:line-break/>C<text:tab/>D</text:p></table:table-cell>`+
			`<table:table-cell office:value-type="string"><text:p>Hello</text:p><office:annotation><text:p>secret</text:p></office:annotation></table:table-cell>`+
			`<table:table-cell office:value-type="string"><text:p>caf&#233;</text:p></table:table-cell>`,
	)))
	assertRows(t, doc.Tabs[0].Data.Rows, [][]string{{
		"Rich Text\nLine 2",
		"A   B\nC\tD",
		"Hello",
		"caf\u00e9",
	}})
}

func TestConvertToQsheetRepeatsAndMerges(t *testing.T) {
	doc, _ := convertSheet(t, sheet("R",
		row(floatCell("1", "1")+
			`<table:table-cell table:number-columns-repeated="2"/>`+
			floatCell("2", "2"))+
			row(`<table:table-cell office:value-type="string" table:number-columns-repeated="3"><text:p>x</text:p></table:table-cell>`)+
			row(strCell("top"))+
			`<table:table-row table:number-rows-repeated="2"/>`+
			row(strCell("bottom"))+
			row(`<table:table-cell office:value-type="string" table:number-columns-spanned="2"><text:p>Hi</text:p></table:table-cell>`+
				`<table:covered-table-cell/>`+
				floatCell("3", "3"))+
			`<table:table-row><table:covered-table-cell/><table:table-cell office:value-type="string"><text:p>S</text:p></table:table-cell></table:table-row>`,
	))
	assertRows(t, doc.Tabs[0].Data.Rows, [][]string{
		{"1", "", "", "2"},
		{"x", "x", "x", ""},
		{"top", "", "", ""},
		{"", "", "", ""},
		{"", "", "", ""},
		{"bottom", "", "", ""},
		{"Hi", "", "3", ""},
		{"", "S", "", ""},
	})
}

func TestConvertToQsheetDropsTrailingRepeats(t *testing.T) {
	doc, _ := convertSheet(t, sheet("Edge",
		row(strCell("A")+`<table:table-cell table:number-columns-repeated="1000000000"/>`)+
			row(strCell("B"))+
			`<table:table-row table:number-rows-repeated="999999999999999999999"><table:table-cell table:number-columns-repeated="999999999999999999999"/></table:table-row>`+
			row(`<table:table-cell><text:p>   </text:p></table:table-cell>`),
	))
	assertRows(t, doc.Tabs[0].Data.Rows, [][]string{{"A"}, {"B"}})
}

func TestConvertToQsheetGroupsAndColumns(t *testing.T) {
	doc, _ := convertSheet(t, `<table:table table:name="G">`+
		`<table:table-column table:number-columns-repeated="99"/>`+
		`<table:table-header-rows>`+row(strCell("H"))+`</table:table-header-rows>`+
		`<table:table-row-group>`+row(floatCell("1", "1"))+`</table:table-row-group>`+
		`</table:table>`)
	assertRows(t, doc.Tabs[0].Data.Rows, [][]string{{"H"}, {"1"}})
}

func TestConvertToQsheetSheets(t *testing.T) {
	doc, result := convertSheet(t,
		sheet("First", row(strCell("a")))+
			`<table:table>`+row(strCell("b"))+`</table:table>`+
			`<table:table table:name="Empty"/>`)
	if len(doc.Tabs) != 3 {
		t.Fatalf("tabs = %+v", doc.Tabs)
	}
	if doc.Tabs[0].Name != "First" || doc.Tabs[1].Name != "Sheet2" || doc.Tabs[2].Name != "Empty" {
		t.Fatalf("names = %q %q %q", doc.Tabs[0].Name, doc.Tabs[1].Name, doc.Tabs[2].Name)
	}
	assertRows(t, doc.Tabs[0].Data.Rows, [][]string{{"a"}})
	assertRows(t, doc.Tabs[1].Data.Rows, [][]string{{"b"}})
	assertRows(t, doc.Tabs[2].Data.Rows, [][]string{{""}})
	if result.Tabs != 3 || result.Rows != 3 || result.Cells != 2 {
		t.Fatalf("result = %+v", result)
	}
}

func TestConvertToQsheetEnvelopeBytes(t *testing.T) {
	data, err := packageBytes(contentXML(sheet("A&amp;B", row(strCell("a&quot;b")))), mimeSpreadsheet, "")
	if err != nil {
		t.Fatal(err)
	}
	_, _, raw := convertPackage(t, data)
	const want = `{"tabs":[{"name":"A\u0026B","data":{"rows":[["a\"b"]]}}]}`
	if raw != want {
		t.Fatalf("envelope = %s, want %s", raw, want)
	}
}

func TestConvertToQsheetTemplateAndIgnoredParts(t *testing.T) {
	data, err := packageBytes(contentXML(sheet("S", row(strCell("keep")))), mimeTemplate, "",
		entry{name: "Basic/script.xml", body: "<script>ignored</script>"})
	if err != nil {
		t.Fatal(err)
	}
	doc, _, raw := convertPackage(t, data)
	assertRows(t, doc.Tabs[0].Data.Rows, [][]string{{"keep"}})
	if strings.Contains(raw, "ignored") {
		t.Fatalf("script text leaked into the envelope: %s", raw)
	}
}

func TestConvertToQsheetRejects(t *testing.T) {
	text, err := packageBytes(contentXML(sheet("S", row(strCell("a")))), "application/vnd.oasis.opendocument.text", "")
	if err != nil {
		t.Fatal(err)
	}
	// packageBytes always writes content.xml, so build one without it.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("mimetype")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, mimeSpreadsheet); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		data []byte
		err  error
	}{
		{"not a zip", []byte("hello"), ErrMalformed},
		{"empty", nil, ErrMalformed},
		{"a text document", text, ErrMalformed},
		{"no content part", buf.Bytes(), ErrMalformed},
		{"malformed content", mustPackage(t, "<not-xml"), ErrMalformed},
		{"no sheet", mustPackage(t, contentXML("")), ErrMalformed},
		{"text body", mustPackage(t, `<?xml version="1.0"?><office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"><office:body><office:text/></office:body></office:document-content>`), ErrMalformed},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			size := int64(len(tt.data))
			if tt.name == "empty" {
				size = 0
			}
			var out bytes.Buffer
			_, err := ConvertToQsheet(ConvertToQsheetParams{Source: bytes.NewReader(tt.data), Size: size, Out: &out})
			if !errors.Is(err, tt.err) {
				t.Fatalf("err = %v, want %v", err, tt.err)
			}
		})
	}
	for _, repeat := range []string{"0", "-1", "abc", "1.5"} {
		t.Run("repeat "+repeat, func(t *testing.T) {
			wantContentErr(t, sheet("S", `<table:table-row table:number-rows-repeated="`+repeat+`">`+strCell("a")+`</table:table-row>`), ErrMalformed)
		})
	}
}

func TestConvertToQsheetEncrypted(t *testing.T) {
	enc := manifestXML(mimeSpreadsheet, "")
	enc = strings.Replace(enc,
		`<manifest:file-entry manifest:full-path="content.xml" manifest:media-type="text/xml"/>`,
		`<manifest:file-entry manifest:full-path="content.xml" manifest:media-type="text/xml"><manifest:encryption-data/></manifest:file-entry>`,
		1)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, p := range []entry{
		{name: "mimetype", body: mimeSpreadsheet},
		{name: "META-INF/manifest.xml", body: enc},
		{name: "content.xml", body: contentXML(sheet("S", row(strCell("secret"))))},
	} {
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
	wantErr(t, buf.Bytes(), ErrEncrypted)
}

func TestConvertToQsheetLimits(t *testing.T) {
	wantContentErr(t, sheet("S", `<table:table-row table:number-rows-repeated="1000000000">`+strCell("a")+`</table:table-row>`), ErrTooLarge)
	wantContentErr(t, sheet("S", row(`<table:table-cell office:value-type="float" office:value="1" table:number-columns-repeated="16385"/>`)), ErrTooLarge)
	wantContentErr(t, sheet("S",
		row(floatCell("1", "1")+`<table:table-cell table:number-columns-repeated="1000000000"/>`+floatCell("2", "2"))), ErrTooLarge)
	wantContentErr(t, sheet("S",
		`<table:table-row table:number-rows-repeated="1000000000"/>`+row(strCell("a"))), ErrTooLarge)
	wantContentErr(t, sheet("S", `<table:table-row table:number-rows-repeated="10001">`+
		`<table:table-cell office:value-type="float" office:value="1" table:number-columns-repeated="100"/>`+
		`</table:table-row>`), ErrTooLarge)

	// A long string repeated across a legal grid is an output bomb, not a
	// cell-count bomb: 16384 cells is under MaxCells.
	wide := strings.Repeat("a", 20_000)
	wantContentErr(t, sheet("S", row(
		`<table:table-cell office:value-type="string" table:number-columns-repeated="16384"><text:p>`+wide+`</text:p></table:table-cell>`,
	)), ErrTooLarge)

	wantContentErr(t, sheet("S", row(
		`<table:table-cell office:value-type="string"><text:p>A<text:s text:c="1000001"/>B</text:p></table:table-cell>`,
	)), ErrTooLarge)

	names := ""
	for i := 0; i < MaxSheets+1; i++ {
		names += `<table:table table:name="S"/>`
	}
	wantContentErr(t, names, ErrTooLarge)

	second := `<table:table table:name="Full"><table:table-row table:number-rows-repeated="100000">` + strCell("a") + `</table:table-row></table:table>` +
		sheet("More", row(strCell("b")))
	wantContentErr(t, second, ErrTooLarge)

	deep := sheet("S", row(`<table:table-cell office:value-type="string"><text:p>`+
		strings.Repeat("<text:span>", 300)+"x"+strings.Repeat("</text:span>", 300)+
		`</text:p></table:table-cell>`))
	wantContentErr(t, deep, ErrTooLarge)
	if !errors.Is(ErrTooLarge, officezip.ErrTooLarge) {
		t.Fatal("ErrTooLarge is not the officezip sentinel")
	}
}

func TestConvertToQsheetAcceptsTheRowLimit(t *testing.T) {
	doc, result := convertSheet(t, sheet("S", `<table:table-row table:number-rows-repeated="100000">`+strCell("a")+`</table:table-row>`))
	if len(doc.Tabs) != 1 || len(doc.Tabs[0].Data.Rows) != MaxRows {
		t.Fatalf("rows = %d", len(doc.Tabs[0].Data.Rows))
	}
	if doc.Tabs[0].Data.Rows[0][0] != "a" || doc.Tabs[0].Data.Rows[MaxRows-1][0] != "a" {
		t.Fatal("repeated value did not fill the sheet")
	}
	if result.Rows != MaxRows || result.Cells != MaxRows {
		t.Fatalf("result = %+v", result)
	}
}

func TestAccountGridBoundary(t *testing.T) {
	got, err := accountGrid(0, 100_000, 10)
	if err != nil || got != 1_000_000 {
		t.Fatalf("accountGrid = %d, %v", got, err)
	}
	if _, err := accountGrid(900_000, 10_000, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := accountGrid(900_001, 10_000, 10); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v", err)
	}
	if _, err := accountGrid(0, 100_000, 11); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v", err)
	}
}

func TestTrailingRepeatIsNotExpanded(t *testing.T) {
	inner := sheet("S", row(strCell("A"))+
		`<table:table-row table:number-rows-repeated="1000000000"><table:table-cell table:number-columns-repeated="1000000000"/></table:table-row>`)
	data, err := packageBytes(contentXML(inner), mimeSpreadsheet, "")
	if err != nil {
		t.Fatal(err)
	}
	var doc qsheetDoc
	alloc := allocated(func() {
		var out bytes.Buffer
		if _, err := ConvertToQsheet(ConvertToQsheetParams{Source: bytes.NewReader(data), Size: int64(len(data)), Out: &out}); err != nil {
			t.Errorf("ConvertToQsheet: %v", err)
			return
		}
		if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
			t.Errorf("json: %v", err)
		}
	})
	if t.Failed() {
		return
	}
	assertRows(t, doc.Tabs[0].Data.Rows, [][]string{{"A"}})
	if alloc > 32<<20 {
		t.Fatalf("trailing repeats allocated %d bytes", alloc)
	}
}

func TestEntitiesAreNotExpanded(t *testing.T) {
	body := `<?xml version="1.0"?>` +
		`<!DOCTYPE d [<!ENTITY x SYSTEM "file:///etc/passwd">]>` +
		contentXML(sheet("S", row(`<table:table-cell office:value-type="string"><text:p>&x;</text:p></table:table-cell>`)))
	data, err := packageBytes(body, mimeSpreadsheet, "")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	_, err = ConvertToQsheet(ConvertToQsheetParams{Source: bytes.NewReader(data), Size: int64(len(data)), Out: &out})
	if err == nil && strings.Contains(out.String(), "root:") {
		t.Fatal("an external entity was expanded")
	}
	if err == nil {
		t.Fatal("expected the entity to be rejected")
	}
	if !errors.Is(err, ErrMalformed) {
		t.Fatalf("err = %v, want ErrMalformed", err)
	}
}

func TestConvertToQsheetCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	data, err := packageBytes(contentXML(sheet("S", row(strCell("a")))), mimeSpreadsheet, "")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	_, err = ConvertToQsheet(ConvertToQsheetParams{Ctx: ctx, Source: bytes.NewReader(data), Size: int64(len(data)), Out: &out})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("wrote %q", out.String())
	}
}

func TestConvertToQsheetRequiresSource(t *testing.T) {
	_, err := ConvertToQsheet(ConvertToQsheetParams{})
	if err == nil {
		t.Fatal("expected an error")
	}
}

func mustPackage(t testing.TB, content string) []byte {
	t.Helper()
	data, err := packageBytes(content, mimeSpreadsheet, "")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func allocated(fn func()) uint64 {
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	fn()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}
