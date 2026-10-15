package docreadutil

// cspell:ignore applet andi infilter monoface soffice

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"os"
	"slices"
	"strings"
	"testing"
)

// The namespaces a test ODT part declares on its root.
const odtNamespaces = `xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" ` +
	`xmlns:style="urn:oasis:names:tc:opendocument:xmlns:style:1.0" ` +
	`xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0" ` +
	`xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0" ` +
	`xmlns:draw="urn:oasis:names:tc:opendocument:xmlns:drawing:1.0" ` +
	`xmlns:fo="urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0" ` +
	`xmlns:svg="urn:oasis:names:tc:opendocument:xmlns:svg-compatible:1.0" ` +
	`xmlns:script="urn:oasis:names:tc:opendocument:xmlns:script:1.0" ` +
	`xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:dc="http://purl.org/dc/elements/1.1/" ` +
	`xmlns:meta="urn:oasis:names:tc:opendocument:xmlns:meta:1.0" ` +
	`xmlns:manifest="urn:oasis:names:tc:opendocument:xmlns:manifest:1.0"`

// odtDoc is an ODT package built for a test: each field is a part's
// content inside its container.
type odtDoc struct {
	// body is office:text's content.
	body string
	// automatic is content.xml's office:automatic-styles content.
	automatic string
	// styles is styles.xml's office:styles content, and master its
	// office:master-styles content; styles.xml is left out when both are
	// empty.
	styles string
	master string
	// title is meta.xml's title; meta.xml is left out when it is empty.
	title string
	// manifest are extra manifest:file-entry elements.
	manifest string
	// files are extra entries, by name.
	files map[string]string
}

// zip builds the package.
func (d odtDoc) zip(t testing.TB) []byte {
	t.Helper()
	type file struct{ name, body string }
	files := []file{
		{"mimetype", "application/vnd.oasis.opendocument.text"},
		{"content.xml", `<?xml version="1.0" encoding="UTF-8"?><office:document-content ` + odtNamespaces + `>` +
			`<office:automatic-styles>` + d.automatic + `</office:automatic-styles>` +
			`<office:body><office:text>` + d.body + `</office:text></office:body></office:document-content>`},
		{"META-INF/manifest.xml", `<manifest:manifest ` + odtNamespaces + `>` +
			`<manifest:file-entry manifest:full-path="/" manifest:media-type="application/vnd.oasis.opendocument.text"/>` +
			`<manifest:file-entry manifest:full-path="content.xml" manifest:media-type="text/xml"/>` + d.manifest + `</manifest:manifest>`},
	}
	if d.styles != "" || d.master != "" {
		files = append(files, file{"styles.xml", `<office:document-styles ` + odtNamespaces + `>` +
			`<office:styles>` + d.styles + `</office:styles><office:master-styles>` + d.master + `</office:master-styles></office:document-styles>`})
	}
	if d.title != "" {
		files = append(files, file{"meta.xml", `<office:document-meta ` + odtNamespaces + `><office:meta><dc:title>` + d.title + `</dc:title></office:meta></office:document-meta>`})
	}
	for name, body := range d.files {
		files = append(files, file{name, body})
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range files {
		w, err := zw.Create(f.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(w, f.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// convertOdt runs Convert over an ODT package held in memory.
func convertOdt(t testing.TB, data []byte, output Output, limits Limits) (string, ConvertResult, error) {
	t.Helper()
	var out strings.Builder
	result, err := Convert(ConvertParams{
		Source: bytes.NewReader(data), Size: int64(len(data)), Format: FormatODT,
		Output: output, Out: &out, Limits: limits,
	})
	return out.String(), result, err
}

// readOdtStream converts an ODT package to NDJSON and parses it.
func readOdtStream(t testing.TB, d odtDoc, limits Limits) (stream, ConvertResult) {
	t.Helper()
	out, result, err := convertOdt(t, d.zip(t), OutputNDJSON, limits)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	return parseStream(t, out), result
}

// odtPara is a text:p, styled when style is not empty.
func odtPara(style, content string) string {
	if style != "" {
		return `<text:p text:style-name="` + style + `">` + content + `</text:p>`
	}
	return `<text:p>` + content + `</text:p>`
}

// textStyle is a text style with the given text properties.
func textStyle(name, props string) string {
	return `<style:style style:name="` + name + `" style:family="text"><style:text-properties ` + props + `/></style:style>`
}

// item is a text:list-item holding content.
func item(content string) string { return `<text:list-item>` + content + `</text:list-item>` }

// odtCell is a table:table-cell holding one paragraph.
func odtCell(attrs, text string) string {
	return `<table:table-cell ` + attrs + `>` + odtPara("", text) + `</table:table-cell>`
}

// frame is a draw:frame holding content.
func frame(attrs, content string) string {
	return `<draw:frame ` + attrs + `>` + content + `</draw:frame>`
}

// equalTexts fails the test unless got is want.
func equalTexts(t *testing.T, got []string, want ...string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("blocks:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// equalRuns fails the test unless got is want.
func equalRuns(t *testing.T, got, want []Run) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("runs:\n%+v\nwant:\n%+v", got, want)
	}
}

// testdata/odt/sample.odt was written by LibreOffice from an HTML page, with
// soffice --headless --infilter="HTML (StarWriter)" --convert-to odt:writer8.
func TestOdtWrittenByLibreOffice(t *testing.T) {
	data, err := os.ReadFile("testdata/odt/sample.odt")
	if err != nil {
		t.Fatal(err)
	}
	out, result, err := convertOdt(t, data, OutputNDJSON, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	s := parseStream(t, out)
	equalTexts(t, texts(s.blocks),
		"Quarterly report",
		"Plain text with bold, italic, underlined, struck, x2, H2O and code().",
		"Centered paragraph with a link.",
		"Lists",
		"First bullet", "Second bullet", "Nested bullet",
		"One", "Two", "Two a", "Two b", "Three",
		"Table",
		"table[Wide cell|Tall cell|A|B]",
		"An image: ",
		"image:Pictures/100000000000000400000004F847FE63.png",
		"Body text with a note1.",
		"Notes",
		"1 The note's text.",
	)
	if s.meta.Title != "Sample ODT" || s.meta.Format != FormatODT || result.Title != "Sample ODT" {
		t.Errorf("meta = %+v", s.meta)
	}
	if b := s.blocks[0]; b.Type != BlockHeading || b.Level != 1 || s.blocks[3].Level != 2 {
		t.Errorf("headings = %+v, %+v", b, s.blocks[3])
	}
	equalRuns(t, s.blocks[1].Runs, []Run{{Text: "Plain text with "}, {Text: "bold", Bold: true}, {Text: ", "},
		{Text: "italic", Italic: true}, {Text: ", "}, {Text: "underlined", Underline: true}, {Text: ", "},
		{Text: "struck", Strike: true}, {Text: ", x"}, {Text: "2", Superscript: true}, {Text: ", H"},
		{Text: "2", Subscript: true}, {Text: "O and "}, {Text: "code()", Code: true}, {Text: "."}})
	if b := s.blocks[2]; b.Align != "center" || b.Runs[1].Link != "https://example.com/" {
		t.Errorf("centered paragraph = %+v", b)
	}
	var markers []string
	for _, b := range s.blocks[4:12] {
		markers = append(markers, b.Marker)
		if b.Type != BlockListItem {
			t.Errorf("%q is a %s, want a list item", plain(b), b.Type)
		}
	}
	if got := strings.Join(markers, " "); got != "• • • 1. 2. a. b. 3." {
		t.Errorf("markers = %q", got)
	}
	if s.blocks[6].Level != 1 || s.blocks[9].Level != 1 || !s.blocks[9].Ordered || s.blocks[4].Ordered {
		t.Errorf("list items: %+v %+v %+v", s.blocks[4], s.blocks[6], s.blocks[9])
	}
	table := s.blocks[13]
	if c := table.Rows[0].Cells; len(c) != 2 || c[0].ColSpan != 2 || c[1].RowSpan != 2 || len(table.Rows[1].Cells) != 2 {
		t.Errorf("table = %+v", table)
	}
	if img := s.blocks[15]; img.Alt != "A red dot" || img.Width < 39 || img.Height < 39 {
		t.Errorf("image = %+v", img)
	}
	if s.end.Truncated || len(s.end.Warnings) != 0 {
		t.Errorf("end = %+v", s.end)
	}
}

func TestOdtHeadings(t *testing.T) {
	d := odtDoc{
		styles: `<style:style style:name="Title" style:family="paragraph"/>` +
			`<style:style style:name="Heading_20_2" style:display-name="Heading 2" style:family="paragraph" style:default-outline-level="2"/>` +
			`<style:style style:name="Subtitle" style:family="paragraph" style:parent-style-name="Title"/>`,
		automatic: `<style:style style:name="P1" style:family="paragraph" style:parent-style-name="Heading_20_2"/>`,
		body: `<text:h text:outline-level="3">Three</text:h>` +
			`<text:h text:outline-level="9">Deep</text:h>` +
			`<text:h text:style-name="P1">From its style</text:h>` +
			`<text:h>Bare</text:h>` +
			odtPara("Title", "A title") +
			odtPara("Subtitle", "Based on the title") +
			odtPara("P1", "A Heading 2 paragraph"),
	}
	s, _ := readOdtStream(t, d, Limits{})
	var levels []int
	for _, b := range s.blocks {
		if b.Type != BlockHeading {
			t.Fatalf("%q is a %s", plain(b), b.Type)
		}
		levels = append(levels, b.Level)
	}
	if want := []int{3, 6, 2, 1, 1, 1, 2}; !slices.Equal(levels, want) {
		t.Errorf("levels = %v, want %v", levels, want)
	}
}

func TestOdtRunFormatting(t *testing.T) {
	d := odtDoc{
		styles: `<style:default-style style:family="paragraph"><style:text-properties fo:font-style="italic"/></style:default-style>` +
			`<style:style style:name="Strong" style:family="text"><style:text-properties fo:font-weight="700"/></style:style>` +
			`<style:style style:name="Plain" style:family="paragraph"><style:text-properties fo:font-style="normal"/></style:style>`,
		automatic: textStyle("T1", `fo:font-weight="bold" style:text-underline-style="solid"`) +
			`<style:style style:name="T2" style:family="text" style:parent-style-name="Strong"><style:text-properties style:text-line-through-style="solid"/></style:style>` +
			textStyle("T3", `style:text-position="super 58%"`) +
			textStyle("T4", `style:text-position="-33% 58%"`) +
			textStyle("T5", `fo:font-family="'Courier New', monospace"`) +
			textStyle("T6", `text:display="none"`) +
			textStyle("T7", `fo:font-weight="normal" style:text-underline-style="none"`) +
			textStyle("T8", `style:font-name="Fixed"`),
		body: odtPara("", `default <text:span text:style-name="T1">bu</text:span>`) +
			odtPara("Plain", `<text:span text:style-name="T2">bs</text:span>`+
				`<text:span text:style-name="T3">up</text:span><text:span text:style-name="T4">down</text:span>`+
				`<text:span text:style-name="T5">mono</text:span><text:span text:style-name="T6">hidden</text:span>`+
				`<text:span text:style-name="T8">face</text:span>`+
				`<text:span text:style-name="T1">b<text:span text:style-name="T7">off</text:span></text:span>`),
	}
	// A font face is declared before the styles that name it.
	data := rezipWith(t, d.zip(t), func(n string) bool { return n != "content.xml" }, "content.xml",
		strings.Replace(contentOf(t, d), `<office:automatic-styles>`,
			`<office:font-face-decls><style:font-face style:name="Fixed" style:font-pitch="fixed"/></office:font-face-decls><office:automatic-styles>`, 1))
	out, _, err := convertOdt(t, data, OutputNDJSON, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	s := parseStream(t, out)
	equalRuns(t, s.blocks[0].Runs, []Run{{Text: "default ", Italic: true}, {Text: "bu", Italic: true, Bold: true, Underline: true}})
	equalRuns(t, s.blocks[1].Runs, []Run{{Text: "bs", Bold: true, Strike: true}, {Text: "up", Superscript: true},
		{Text: "down", Subscript: true}, {Text: "monoface", Code: true},
		{Text: "b", Bold: true, Underline: true}, {Text: "off"}})
}

// contentOf is the content.xml a test document builds.
func contentOf(t *testing.T, d odtDoc) string {
	t.Helper()
	data := d.zip(t)
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	rc, err := zr.Open("content.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	content, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func TestOdtWhiteSpace(t *testing.T) {
	d := odtDoc{body: odtPara("", "\n   lead  and\n\ttrail <text:s/>x<text:s text:c=\"3\"/>y<text:tab/>z<text:line-break/> next") +
		odtPara("", `<text:s text:c="99999999"/>`)}
	s, _ := readOdtStream(t, d, Limits{})
	if got := plain(s.blocks[0]); got != "lead and trail  x   y\tz\nnext" {
		t.Errorf("text = %q", got)
	}
	if got := plain(s.blocks[1]); len(got) != maxSpaces {
		t.Errorf("a huge text:s gave %d spaces, want %d", len(got), maxSpaces)
	}
}

// odtListStyles are list styles for the list tests.
const odtListStyles = `<text:list-style style:name="Num">` +
	`<text:list-level-style-number text:level="1" style:num-format="1" style:num-suffix="."/>` +
	`<text:list-level-style-number text:level="2" style:num-format="a" style:num-prefix="(" style:num-suffix=")" text:display-levels="2"/>` +
	`<text:list-level-style-number text:level="3" style:num-format="I" text:start-value="4"/>` +
	`</text:list-style>` +
	`<text:list-style style:name="Bul"><text:list-level-style-bullet text:level="1" text:bullet-char="` + "" + `"/>` +
	`<text:list-level-style-bullet text:level="2" text:bullet-char="–"/></text:list-style>` +
	`<text:list-style style:name="Let"><text:list-level-style-number text:level="1" style:num-format="A"/></text:list-style>` +
	`<text:list-style style:name="Bare"><text:list-level-style-number text:level="1" style:num-format=""/></text:list-style>`

func TestOdtLists(t *testing.T) {
	d := odtDoc{
		styles: odtListStyles + `<style:style style:name="Listed" style:family="paragraph" style:list-style-name="Let"/>`,
		body: `<text:list text:style-name="Num" xml:id="L1">` +
			item(odtPara("", "one")+odtPara("", "more of one")) +
			item(odtPara("", "two")+`<text:list>`+item(odtPara("", "two a"))+item(odtPara("", "two b")+
				`<text:list>`+item(odtPara("", "deep"))+`</text:list>`)+`</text:list>`) +
			`<text:list-header>` + odtPara("", "no number") + `</text:list-header>` +
			`<text:list-item text:start-value="10">` + odtPara("", "ten") + `</text:list-item>` +
			`</text:list>` +
			odtPara("", "between") +
			`<text:list text:style-name="Num" text:continue-numbering="true">` + item(odtPara("", "eleven")) + `</text:list>` +
			`<text:list text:style-name="Num">` + item(odtPara("", "again")) + `</text:list>` +
			`<text:list text:continue-list="L1">` + item(odtPara("", "twelve")) + `</text:list>` +
			`<text:list text:style-name="Bul">` + item(odtPara("", "dot")+`<text:list>`+item(odtPara("", "dash"))+`</text:list>`) + `</text:list>` +
			`<text:list>` + item(odtPara("Listed", "from the paragraph style")) + `</text:list>` +
			`<text:list text:style-name="Bare">` + item(odtPara("", "unnumbered")) + `</text:list>` +
			`<text:list text:style-name="Unknown">` + item(`<text:h text:outline-level="2">heading wins</text:h>`) + item(odtPara("", "undefined")) + `</text:list>` +
			`<text:list>` + item(`<table:table><table:table-row>`+odtCell("", "in a table")+`</table:table-row></table:table>`) + `</text:list>`,
	}
	s, _ := readOdtStream(t, d, Limits{})
	var got []string
	for _, b := range s.blocks {
		desc := strings.Join(texts([]Block{b}), "")
		switch b.Type {
		case BlockListItem:
			kind := "-"
			if b.Ordered {
				kind = "#"
			}
			desc = strings.Repeat("  ", b.Level) + kind + b.Marker + " " + desc
		case BlockHeading:
			desc = "h: " + desc
		}
		got = append(got, desc)
	}
	equalTexts(t, got,
		"#1. one", "more of one", "#2. two", "  #(2.a) two a", "  #(2.b) two b", "    #IV deep",
		"no number", "#10. ten", "between", "#11. eleven", "#1. again", "#12. twelve",
		"-• dot", "  -– dash", "#A from the paragraph style", "- unnumbered", "h: heading wins", "-• undefined",
		"table[in a table]")
}

func TestOdtNumberFormats(t *testing.T) {
	cases := []struct {
		n      int
		format string
		sync   bool
		want   string
	}{
		{1, "1", false, "1"}, {26, "a", false, "z"}, {27, "a", false, "aa"}, {28, "A", false, "AB"},
		{28, "a", true, "bb"}, {4, "i", false, "iv"}, {9, "I", false, "IX"}, {0, "a", false, "0"},
		{5, "١", false, "5"},
	}
	for _, c := range cases {
		if got := odtNumber(c.n, c.format, c.sync); got != c.want {
			t.Errorf("odtNumber(%d, %q, %v) = %q, want %q", c.n, c.format, c.sync, got, c.want)
		}
	}
}

func TestOdtTables(t *testing.T) {
	d := odtDoc{body: `<table:table><table:table-column table:number-columns-repeated="3"/>` +
		`<table:table-header-rows><table:table-row>` + odtCell(`table:number-columns-spanned="2" table:number-rows-spanned="2"`, "big") +
		`<table:covered-table-cell/>` + odtCell("", "c") + `</table:table-row></table:table-header-rows>` +
		`<table:table-row><table:covered-table-cell table:number-columns-repeated="2"/>` + odtCell("", "f") + `</table:table-row>` +
		`<table:table-rows><table:table-row table:number-rows-repeated="2">` + odtCell(`table:number-columns-repeated="3"`, "r") + `</table:table-row></table:table-rows>` +
		`<table:table-row>` + odtCell(`table:number-rows-spanned="99"`, "last") + odtCell(`table:number-columns-spanned="9999"`, "wide") + `</table:table-row>` +
		`</table:table>`}
	s, _ := readOdtStream(t, d, Limits{})
	equalTexts(t, texts(s.blocks), "table[big|c|f|r|r|r|r|r|r|last|wide]")
	rows := s.blocks[0].Rows
	if len(rows) != 5 || rows[0].Cells[0].ColSpan != 2 || rows[0].Cells[0].RowSpan != 2 {
		t.Fatalf("rows = %+v", rows)
	}
	if c := rows[4].Cells; c[0].RowSpan != 1 || c[1].ColSpan != maxSpan {
		t.Errorf("last row = %+v, want its row span cut to the table and its column span bounded", c)
	}
}

func TestOdtRepeatsAreBounded(t *testing.T) {
	d := odtDoc{body: `<table:table><table:table-row table:number-rows-repeated="1000000000">` +
		odtCell(`table:number-columns-repeated="1000000000"`, "") + `</table:table-row></table:table>`}
	s, result := readOdtStream(t, d, Limits{})
	if len(s.blocks) != 1 || !s.end.Truncated || !result.Truncated {
		t.Fatalf("end = %+v, %d blocks", s.end, len(s.blocks))
	}
	if rows := s.blocks[0].Rows; len(rows) > maxRepeat || len(rows[0].Cells) != maxRepeat {
		t.Errorf("%d rows of %d cells, want at most %d of %d", len(rows), len(rows[0].Cells), maxRepeat, maxRepeat)
	}
}

func TestOdtNestedTablesFlattenPastTheDepthLimit(t *testing.T) {
	inner := odtPara("", "deepest")
	for range 5 {
		inner = `<table:table><table:table-rows><table:table-row><table:table-cell>` + inner +
			`</table:table-cell></table:table-row></table:table-rows></table:table>`
	}
	s, _ := readOdtStream(t, odtDoc{body: inner}, Limits{MaxTableDepth: 2})
	equalTexts(t, texts(s.blocks), "table[table[deepest]]")
}

func TestOdtTrackedChangesReadAsAccepted(t *testing.T) {
	d := odtDoc{body: `<text:tracked-changes><text:changed-region text:id="c1"><text:deletion>` + odtPara("", "deleted") +
		`</text:deletion></text:changed-region><text:changed-region text:id="c2"><text:insertion/></text:changed-region></text:tracked-changes>` +
		odtPara("", `kept<text:change text:change-id="c1"/> <text:change-start text:change-id="c2"/>inserted<text:change-end text:change-id="c2"/>`)}
	s, _ := readOdtStream(t, d, Limits{})
	equalTexts(t, texts(s.blocks), "kept inserted")
}

// note is a text:note.
func note(id, class, citation, body string) string {
	return `<text:note text:id="` + id + `" text:note-class="` + class + `"><text:note-citation>` + citation +
		`</text:note-citation><text:note-body>` + body + `</text:note-body></text:note>`
}

func TestOdtNotesFollowTheBody(t *testing.T) {
	d := odtDoc{body: odtPara("", "First"+note("ftn1", "footnote", "1", odtPara("", "Note one"))+" and"+
		note("edn1", "endnote", "i", odtPara("", " Note two")+odtPara("", "continued"))) +
		`<table:table><table:table-row><table:table-cell>` + odtPara("", "cell"+note("ftn2", "footnote", "2", odtPara("", "In a cell"))) +
		`</table:table-cell></table:table-row></table:table>` +
		odtPara("", "no id"+`<text:note><text:note-citation text:label="*">9</text:note-citation><text:note-body>`+odtPara("", "lost")+`</text:note-body></text:note>`) +
		`<text:tracked-changes><text:changed-region><text:deletion>` + odtPara("", note("ftn1", "footnote", "1", odtPara("", "deleted twin"))) +
		`</text:deletion></text:changed-region></text:tracked-changes>`}
	s, result := readOdtStream(t, d, Limits{})
	equalTexts(t, texts(s.blocks), "First1 andi", "table[cell2]", "no id*", "Notes", "1 Note one", "i Note two", "continued", "2 In a cell")
	equalRuns(t, s.blocks[0].Runs, []Run{{Text: "First"}, {Text: "1", Superscript: true}, {Text: " and"}, {Text: "i", Superscript: true}})
	equalRuns(t, s.blocks[4].Runs, []Run{{Text: "1", Superscript: true}, {Text: " Note one"}})
	if len(result.Warnings) != 1 || result.Warnings[0] != warnNotes {
		t.Errorf("warnings = %v", result.Warnings)
	}
}

func TestOdtNotesAreCutByTheLimitToo(t *testing.T) {
	var body strings.Builder
	for i := range 20 {
		id := string(rune('a' + i))
		body.WriteString(odtPara("", "x"+note(id, "footnote", id, odtPara("", "note"))))
	}
	s, _ := readOdtStream(t, odtDoc{body: body.String()}, Limits{MaxBlocks: 25})
	if len(s.blocks) != 25 || !s.end.Truncated || plain(s.blocks[20]) != "Notes" {
		t.Errorf("%d blocks, end %+v", len(s.blocks), s.end)
	}
}

func TestOdtImagesAndObjects(t *testing.T) {
	img := func(href string) string { return `<draw:image xlink:href="` + href + `"/>` }
	d := odtDoc{
		manifest: `<manifest:file-entry manifest:full-path="Object 1/" manifest:media-type="application/vnd.oasis.opendocument.chart"/>` +
			`<manifest:file-entry manifest:full-path="Object 2/" manifest:media-type="application/vnd.oasis.opendocument.formula"/>`,
		files: map[string]string{"Pictures/a.png": "png", "Object 1/content.xml": "<x/>"},
		body: odtPara("", "before"+frame(`svg:width="1in" svg:height="2.54cm"`, img("Pictures/a.png")+`<svg:title>Title</svg:title><svg:desc>Desc</svg:desc>`)+"after") +
			frame(`text:anchor-type="page"`, img("./Pictures/a.png")) +
			odtPara("", frame("", img("https://example.com/x.png")+`<svg:desc>remote</svg:desc>`)) +
			odtPara("", frame("", img("../outside.png"))+frame("", img("/etc/passwd"))+frame("", img("file:///etc/passwd"))) +
			odtPara("", frame("", img("Pictures/missing.png"))+frame("", `<draw:image><office:binary-data>aGk=</office:binary-data></draw:image>`)) +
			odtPara("", frame("", `<draw:object xlink:href="./Object 1"/>`+img("./ObjectReplacements/Object 1"))+
				frame("", `<draw:object xlink:href="./Object 2"/>`)+frame("", `<draw:object-ole xlink:href="./Object 3"/>`)+
				frame("", `<draw:plugin xlink:href="https://example.com/applet"/>`)+frame("", `<draw:floating-frame xlink:href="https://example.com/"/>`)) +
			odtPara("", `<draw:custom-shape><text:p>shape text</text:p></draw:custom-shape><draw:g><draw:rect/></draw:g>`) +
			`<draw:line/>` +
			odtPara("", "pic: "+frame("", `<draw:text-box>`+odtPara("", frame("", img("Pictures/a.png")))+odtPara("", "Caption")+`</draw:text-box>`)+" after the box"),
	}
	s, result := readOdtStream(t, d, Limits{})
	equalTexts(t, texts(s.blocks),
		"before", "image:Pictures/a.png", "after",
		"image:Pictures/a.png",
		"image:",
		"image:", "image:", "image:",
		"omitted:image", "omitted:image",
		"omitted:chart", "omitted:equation", "omitted:embedded object", "omitted:embedded object", "omitted:embedded document",
		"omitted:shape", "omitted:shape",
		"omitted:shape",
		"pic: ", "image:Pictures/a.png", "Caption", "after the box",
	)
	if b := s.blocks[1]; b.Alt != "Desc" || b.Width != 96 || b.Height != 96 {
		t.Errorf("image = %+v", b)
	}
	if b := s.blocks[4]; !b.External || b.Part != "" || b.Alt != "remote" {
		t.Errorf("linked image = %+v", b)
	}
	want := []string{"4 linked images not shown", "2 images not shown", "1 chart not shown", "1 equation not shown",
		"2 embedded objects not shown", "1 embedded document not shown", "3 shapes not shown"}
	if !slices.Equal(result.Warnings, want) {
		t.Errorf("warnings = %q, want %q", result.Warnings, want)
	}
}

func TestOdtLinks(t *testing.T) {
	d := odtDoc{body: odtPara("", `<text:a xlink:href="https://example.com/a">web</text:a> `+
		`<text:a xlink:href="mailto:a@example.com">mail</text:a> <text:a xlink:href="javascript:alert(1)">js</text:a> `+
		`<text:a xlink:href="#Heading">inside</text:a> <text:a xlink:href="file:///etc/passwd">file</text:a>`)}
	s, _ := readOdtStream(t, d, Limits{})
	var links []string
	for _, r := range s.blocks[0].Runs {
		if r.Link != "" {
			links = append(links, r.Text+"="+r.Link)
		}
	}
	if want := []string{"web=https://example.com/a", "mail=mailto:a@example.com"}; !slices.Equal(links, want) {
		t.Errorf("links = %q, want %q", links, want)
	}
}

func TestOdtFieldsSectionsAndIndexes(t *testing.T) {
	d := odtDoc{body: `<text:sequence-decls><text:sequence-decl text:name="Figure"/></text:sequence-decls>` +
		`<text:table-of-content><text:table-of-content-source><text:index-title-template>Template</text:index-title-template></text:table-of-content-source>` +
		`<text:index-body><text:index-title>` + odtPara("", "Contents") + `</text:index-title>` + odtPara("", "Intro 1") + `</text:index-body></text:table-of-content>` +
		`<text:section text:name="Shown">` + odtPara("", `Page <text:page-number>3</text:page-number> by <text:author-name>Ann</text:author-name>`) + `</text:section>` +
		`<text:section text:name="Hidden" text:display="none">` + odtPara("", "secret") + `</text:section>` +
		odtPara("", `<text:ruby><text:ruby-base>漢</text:ruby-base><text:ruby-text>kan</text:ruby-text></text:ruby><text:script script:language="js">evil()</text:script>`+
			`<text:hidden-text text:string-value="h"/><text:bookmark text:name="b"/><text:number>7.</text:number>`) +
		`<office:forms/><text:soft-page-break/>`}
	s, _ := readOdtStream(t, d, Limits{})
	equalTexts(t, texts(s.blocks), "Contents", "Intro 1", "Page 3 by Ann", "漢")
}

func TestOdtPageBreaks(t *testing.T) {
	d := odtDoc{
		automatic: `<style:style style:name="Before" style:family="paragraph"><style:paragraph-properties fo:break-before="page"/></style:style>` +
			`<style:style style:name="After" style:family="paragraph"><style:paragraph-properties fo:break-after="page"/></style:style>`,
		body: odtPara("", "one") + odtPara("Before", "two") + odtPara("After", "three") + odtPara("", "four") +
			`<table:table><table:table-row>` + odtCell("", "") + `</table:table-row></table:table>` +
			`<table:table><table:table-row><table:table-cell>` + odtPara("Before", "in a cell") + `</table:table-cell></table:table-row></table:table>`,
	}
	s, _ := readOdtStream(t, d, Limits{})
	equalTexts(t, texts(s.blocks), "one", "pageBreak:", "two", "three", "pageBreak:", "four", "table[]", "table[in a cell]")
}

func TestOdtHeadersFootersAndCommentsAreNamed(t *testing.T) {
	d := odtDoc{
		master: `<style:master-page style:name="Standard"><style:header>` + odtPara("", "head") + `</style:header><style:footer-left/></style:master-page>`,
		body:   odtPara("", `text<office:annotation><dc:creator>Ann</dc:creator>`+odtPara("", "a comment")+`</office:annotation>`),
	}
	s, result := readOdtStream(t, d, Limits{})
	equalTexts(t, texts(s.blocks), "text")
	if !slices.Equal(result.Warnings, []string{warnHeadersFooters, warnComments}) {
		t.Errorf("warnings = %q", result.Warnings)
	}
}

func TestOdtAlignmentAndCounts(t *testing.T) {
	d := odtDoc{
		styles: `<style:style style:name="Centered" style:family="paragraph"><style:paragraph-properties fo:text-align="center"/></style:style>`,
		automatic: `<style:style style:name="P1" style:family="paragraph" style:parent-style-name="Centered"><style:paragraph-properties fo:text-align="start"/></style:style>` +
			`<style:style style:name="P2" style:family="paragraph"><style:paragraph-properties fo:text-align="end"/></style:style>` +
			`<style:style style:name="P3" style:family="paragraph"><style:paragraph-properties fo:text-align="justify"/></style:style>`,
		body: odtPara("Centered", "a b") + odtPara("P1", "c") + odtPara("P2", "d") + odtPara("P3", "e") + odtPara("", ""),
	}
	s, result := readOdtStream(t, d, Limits{})
	var aligns []string
	for _, b := range s.blocks {
		aligns = append(aligns, b.Align)
	}
	if got := strings.Join(aligns, ","); got != "center,,right,justify," {
		t.Errorf("alignments = %q", got)
	}
	if result.Words != 5 || result.Paragraphs != 4 || result.Characters != 6 || len(s.blocks) != 5 {
		t.Errorf("result = %+v", result)
	}
}

func TestOdtStyleChainsEnd(t *testing.T) {
	d := odtDoc{
		styles: `<style:style style:name="A" style:family="paragraph" style:parent-style-name="B"><style:text-properties fo:font-weight="bold"/></style:style>` +
			`<style:style style:name="B" style:family="paragraph" style:parent-style-name="A"/>`,
		automatic: `<style:style style:name="A" style:family="paragraph" style:parent-style-name="A"><style:text-properties fo:font-style="italic"/></style:style>` +
			`<style:style style:name="Wide" style:family="table"/>`,
		body: odtPara("A", "loops") + odtPara("B", "also"),
	}
	s, _ := readOdtStream(t, d, Limits{})
	equalRuns(t, s.blocks[0].Runs, []Run{{Text: "loops", Bold: true, Italic: true}})
	equalTexts(t, texts(s.blocks), "loops", "also")
}

func TestOdtDelta(t *testing.T) {
	d := odtDoc{body: `<text:h text:outline-level="2">Head</text:h>` + `<text:list>` + item(odtPara("", "item")) + `</text:list>`}
	out, result, err := convertOdt(t, d.zip(t), OutputDelta, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `{"insert":"\n","attributes":{"header":2}}`) || !strings.Contains(out, `"list":"bullet"`) || result.Blocks != 2 {
		t.Errorf("delta = %s", out)
	}
}

func TestOdtLimits(t *testing.T) {
	many := strings.Repeat(textStyle("T", `fo:font-weight="bold"`), 30)
	if _, _, err := convertOdt(t, odtDoc{body: odtPara("", "x"), automatic: many}.zip(t), OutputNDJSON, Limits{MaxDefinitions: 20}); !errors.Is(err, ErrTooLarge) {
		t.Errorf("too many styles: err = %v, want ErrTooLarge", err)
	}
	long := textStyle(strings.Repeat("n", 2000), "")
	if _, _, err := convertOdt(t, odtDoc{body: odtPara("", "x"), automatic: long}.zip(t), OutputNDJSON, Limits{MaxHeldBytes: 1000}); !errors.Is(err, ErrTooLarge) {
		t.Errorf("long style names: err = %v, want ErrTooLarge", err)
	}
	big := odtDoc{body: odtPara("", "x"), styles: `<style:style style:name="S" style:family="paragraph"/>` + strings.Repeat("<!-- padding -->", 1000)}
	if _, _, err := convertOdt(t, big.zip(t), OutputNDJSON, Limits{MaxHeldBytes: 4096}); !errors.Is(err, ErrTooLarge) {
		t.Errorf("large styles.xml: err = %v, want ErrTooLarge", err)
	}
	s, result := readOdtStream(t, odtDoc{body: strings.Repeat(odtPara("", "word"), 100)}, Limits{MaxBlocks: 10})
	if len(s.blocks) != 10 || !result.Truncated {
		t.Errorf("block limit: %d blocks, %+v", len(s.blocks), result)
	}
	deep := strings.Repeat("<text:span>", 300) + "x" + strings.Repeat("</text:span>", 300)
	if _, _, err := convertOdt(t, odtDoc{body: odtPara("", deep)}.zip(t), OutputNDJSON, Limits{}); !errors.Is(err, ErrTooLarge) {
		t.Errorf("deep nesting: err = %v, want ErrTooLarge", err)
	}
}

func TestOdtReadingStopsAtTheLimit(t *testing.T) {
	data := odtDoc{body: strings.Repeat(odtPara("", strings.Repeat("words and more words ", 50)), 20000)}.zip(t)
	var result ConvertResult
	var err error
	heap := allocated(func() {
		result, err = Convert(ConvertParams{Source: bytes.NewReader(data), Size: int64(len(data)), Format: FormatODT, Out: io.Discard,
			Limits: Limits{MaxOutputBytes: 1 << 20}})
	})
	if err != nil || !result.Truncated {
		t.Fatalf("result %+v, err %v", result, err)
	}
	// About 20 MB of text lies past a 1 MiB limit; none of it is read.
	if heap > 32<<20 {
		t.Errorf("allocated %d MiB reading past a 1 MiB limit", heap>>20)
	}
}

func TestOdtMalformedDocuments(t *testing.T) {
	good := odtDoc{body: odtPara("", "x"), styles: `<style:style style:name="S" style:family="paragraph"/>`}
	replace := func(name, body string) []byte {
		return rezipWith(t, good.zip(t), func(n string) bool { return n != name }, name, body)
	}
	cases := map[string][]byte{
		"not a zip":    []byte("plain text"),
		"no content":   rezip(t, good.zip(t), func(n string) bool { return n != "content.xml" }),
		"a docx":       testDoc{body: para("x")}.zip(t),
		"wrong root":   replace("content.xml", `<office:document-styles xmlns:office="urn:x"/>`),
		"spreadsheet":  replace("content.xml", `<office:document-content xmlns:office="urn:x"><office:body><office:spreadsheet/></office:body></office:document-content>`),
		"bad xml":      replace("content.xml", `<office:document-content xmlns:office="urn:x" xmlns:text="urn:t"><office:body><office:text><text:p>x</office:text>`),
		"unclosed":     replace("content.xml", `<office:document-content xmlns:office="urn:x"><office:body><office:text>`),
		"bad styles":   replace("styles.xml", `<office:document-styles><office:styles><style:style>`),
		"bad manifest": replace("META-INF/manifest.xml", `<manifest:manifest><manifest:file-entry`),
		"encrypted": replace("META-INF/manifest.xml", `<manifest:manifest xmlns:manifest="urn:m"><manifest:file-entry manifest:full-path="content.xml" manifest:media-type="text/xml">`+
			`<manifest:encryption-data manifest:checksum="x"/></manifest:file-entry></manifest:manifest>`),
	}
	for name, data := range cases {
		out, _, err := convertOdt(t, data, OutputNDJSON, Limits{})
		if !errors.Is(err, ErrMalformed) {
			t.Errorf("%s: err = %v, want ErrMalformed", name, err)
		}
		if out != "" {
			t.Errorf("%s: wrote %q before failing", name, out)
		}
	}
}

func TestOdtWithoutManifestStylesOrMetaIsRead(t *testing.T) {
	data := rezip(t, odtDoc{body: odtPara("", "bare")}.zip(t), func(n string) bool { return n == "content.xml" })
	out, _, err := convertOdt(t, data, OutputNDJSON, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	equalTexts(t, texts(parseStream(t, out).blocks), "bare")
}

func TestOdtAFailureMidwayLeavesNoEndLine(t *testing.T) {
	data := rezipWith(t, odtDoc{body: odtPara("", "x")}.zip(t), func(n string) bool { return n != "content.xml" }, "content.xml",
		`<office:document-content xmlns:office="urn:o" xmlns:text="urn:t"><office:body><office:text>`+
			strings.Repeat(`<text:p>fine</text:p>`, 3)+`<text:p>broken</text:x>`)
	out, _, err := convertOdt(t, data, OutputNDJSON, Limits{})
	if !errors.Is(err, ErrMalformed) || strings.Contains(out, `"type":"end"`) {
		t.Errorf("err = %v, out:\n%s", err, out)
	}
}

func TestOdtFormatForName(t *testing.T) {
	if f, ok := FormatForName("Report.ODT"); !ok || f != FormatODT {
		t.Errorf("FormatForName = %q, %v", f, ok)
	}
}

// odtFuzzBodies are ODT bodies that reach every part of the reader.
var odtFuzzBodies = []string{
	odtPara("P1", `Hello <text:span text:style-name="T1">world</text:span><text:s text:c="2"/><text:tab/><text:line-break/>`),
	`<text:h text:outline-level="2">H</text:h><text:list text:style-name="Num" xml:id="L">` + item(odtPara("", "a")+`<text:list>`+item(odtPara("", "b"))+`</text:list>`) +
		`</text:list><text:list text:continue-list="L"><text:list-item text:start-value="9">` + odtPara("", "c") + `</text:list-item></text:list>`,
	`<table:table><table:table-row table:number-rows-repeated="3">` + odtCell(`table:number-columns-spanned="2" table:number-rows-spanned="2" table:number-columns-repeated="2"`, "x") +
		`<table:covered-table-cell/></table:table-row></table:table>`,
	odtPara("", "n"+note("n1", "footnote", "1", odtPara("", "body"))+`<text:a xlink:href="https://example.com">l</text:a>`+
		frame(`svg:width="1cm"`, `<draw:image xlink:href="Pictures/a.png"/>`)+frame("", `<draw:text-box>`+odtPara("", "cap")+`</draw:text-box>`)+
		frame("", `<draw:object xlink:href="./Object 1"/>`)+`<draw:custom-shape/><office:annotation/>`),
	`<text:section>` + odtPara("P2", "s") + `</text:section><text:tracked-changes/><text:table-of-content><text:index-body>` + odtPara("", "i") + `</text:index-body></text:table-of-content>`,
}

const odtFuzzStyles = `<style:style style:name="Heading" style:family="paragraph"><style:paragraph-properties fo:break-before="page" fo:text-align="center"/></style:style>` +
	odtListStyles

var odtFuzzAutomatic = `<style:style style:name="P1" style:family="paragraph" style:parent-style-name="Heading"/>` +
	`<style:style style:name="P2" style:family="paragraph" style:list-style-name="Num"><style:paragraph-properties fo:break-after="page"/></style:style>` +
	textStyle("T1", `fo:font-weight="bold" style:text-position="super"`)

// odtFuzzFiles are the parts a fuzzed body may refer to.
var odtFuzzFiles = map[string]string{"Pictures/a.png": "png"}

// FuzzConvertOdt reads arbitrary bytes as an ODT package.
func FuzzConvertOdt(f *testing.F) {
	for _, body := range odtFuzzBodies {
		f.Add(odtDoc{body: body, styles: odtFuzzStyles, automatic: odtFuzzAutomatic, files: odtFuzzFiles}.zip(f))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		readFormatWithoutRecover(t, data, OutputNDJSON, FormatODT, readOdt)
		readFormatWithoutRecover(t, data, OutputDelta, FormatODT, readOdt)
	})
}

// FuzzOdtContentXML reads arbitrary body, style and automatic style XML in
// a well-formed package, where the reader's own logic meets hostile input
// rather than the zip layer.
func FuzzOdtContentXML(f *testing.F) {
	for _, body := range odtFuzzBodies {
		f.Add(body, odtFuzzStyles, odtFuzzAutomatic)
	}
	f.Fuzz(func(t *testing.T, body, styles, automatic string) {
		data := odtDoc{body: body, styles: styles, automatic: automatic, files: odtFuzzFiles}.zip(t)
		readFormatWithoutRecover(t, data, OutputNDJSON, FormatODT, readOdt)
		readFormatWithoutRecover(t, data, OutputDelta, FormatODT, readOdt)
	})
}
