package docreadutil

// cspell:ignore pntxtb ansicpg bkmkstart cellx chftn clmgf clmrg clvmgf clvmrg colortbl datastore emdash enspace fcharset fldinst fldrslt fonttbl footerf fprq ftnsep ilvl intbl itap ldblquote listtext nestcell nestrow nesttableprops nonesttables nonshppict nosupersub outlinelevel pagebb pngblip pnlvlblt pntext rdblquote rtfx shpinst shppict shprslt striked stylesheet themedata trleft trowd uldb ulnone unknownword verylongcontrolwordthatgoespastthirtytwoletters xe écafé привет

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// convertRTF converts an RTF file.
func convertRTF(t testing.TB, src string, output Output, limits Limits) (string, ConvertResult, error) {
	t.Helper()
	var out bytes.Buffer
	result, err := Convert(ConvertParams{Source: strings.NewReader(src), Size: int64(len(src)), Format: FormatRTF,
		Output: output, Out: &out, Limits: limits})
	return out.String(), result, err
}

func TestRTFFormatForName(t *testing.T) {
	if f, ok := FormatForName("Letter.RTF"); !ok || f != FormatRTF {
		t.Errorf("FormatForName = %q, %v", f, ok)
	}
}

// readRTFStream reads an RTF file to NDJSON and parses the stream.
func readRTFStream(t testing.TB, src string, limits Limits) (stream, ConvertResult) {
	t.Helper()
	out, result, err := convertRTF(t, src, OutputNDJSON, limits)
	if err != nil {
		t.Fatalf("readRTF: %v", err)
	}
	return parseStream(t, out), result
}

// rtfDoc is an RTF document around body, with a font table whose f0 is
// Times New Roman, f1 Symbol, f2 Courier New and f3 a Cyrillic font.
func rtfDoc(body string) string {
	return `{\rtf1\ansi\ansicpg1252\deff0{\fonttbl{\f0\froman\fcharset0 Times New Roman;}{\f1\fcharset2 Symbol;}` +
		`{\f2\fmodern\fprq1\fcharset0 Courier New;}{\f3\fcharset204 Arial Cyr;}}` + "\n" + body + `}`
}

func readTestdata(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "rtf", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestRTFLibreOfficeReport(t *testing.T) {
	s, result := readRTFStream(t, readTestdata(t, "libreoffice-report.rtf"), Limits{})
	if s.meta.Title != "Quarterly report" || s.meta.Format != FormatRTF {
		t.Errorf("meta = %+v", s.meta)
	}
	want := []string{
		"Summary", "Revenue rose again, under struck x2 H2O and code(). See the site.", "Details",
		"Centered — “quoted” café Привет 日本語 😀", "First bullet", "Second bullet", "Nested bullet", "One", "Two",
		"table[Wide|Tall|a|b/table[inner 1|inner 2]]", "After the table.",
	}
	if got := texts(s.blocks); !reflect.DeepEqual(got, want) {
		t.Fatalf("blocks = %q", got)
	}
	if b := s.blocks[0]; b.Type != BlockHeading || b.Level != 1 {
		t.Errorf("first block = %+v", b)
	}
	if b := s.blocks[2]; b.Type != BlockHeading || b.Level != 2 {
		t.Errorf("third block = %+v", b)
	}
	runs := s.blocks[1].Runs
	wantRuns := map[string]Run{
		"rose": {Text: "rose", Bold: true}, "again": {Text: "again", Italic: true},
		"under": {Text: "under", Underline: true}, "struck": {Text: "struck", Strike: true},
		"code()":   {Text: "code()", Code: true},
		"the site": {Text: "the site", Underline: true, Link: "https://example.com/q3"},
	}
	found := 0
	for _, r := range runs {
		if w, ok := wantRuns[r.Text]; ok {
			found++
			if r != w {
				t.Errorf("run %q = %+v, want %+v", r.Text, r, w)
			}
		}
		if r.Text == "2" && !r.Superscript && !r.Subscript {
			t.Errorf("a raised or lowered 2 is neither: %+v", r)
		}
	}
	if found != len(wantRuns) {
		t.Errorf("found %d of %d formatted runs in %+v", found, len(wantRuns), runs)
	}
	if s.blocks[3].Align != "center" {
		t.Errorf("centered paragraph = %+v", s.blocks[3])
	}
	for i, want := range []struct {
		level   int
		ordered bool
		marker  string
	}{{0, false, "•"}, {0, false, "•"}, {1, false, "•"}, {0, true, "1."}, {0, true, "2."}} {
		b := s.blocks[4+i]
		if b.Type != BlockListItem || b.Level != want.level || b.Ordered != want.ordered || b.Marker != want.marker {
			t.Errorf("list item %d = %+v", i, b)
		}
	}
	table := s.blocks[9]
	if c := table.Rows[0].Cells; c[0].ColSpan != 2 || c[1].RowSpan != 2 || len(table.Rows[1].Cells) != 2 {
		t.Errorf("table spans = %+v", table.Rows)
	}
	if result.Truncated || len(result.Warnings) != 0 {
		t.Errorf("result = %+v", result)
	}
}

func TestRTFLibreOfficeNotes(t *testing.T) {
	s, result := readRTFStream(t, readTestdata(t, "libreoffice-notes.rtf"), Limits{})
	want := []string{"Body text with a note1 and more.", "omitted:image", " after image", "pageBreak:", "Second page", "Notes", "1The note text."}
	if got := texts(s.blocks); !reflect.DeepEqual(got, want) {
		t.Fatalf("blocks = %q", got)
	}
	if !reflect.DeepEqual(result.Warnings, []string{"1 image not shown"}) {
		t.Errorf("warnings = %q", result.Warnings)
	}
}

func TestRTFWordStyle(t *testing.T) {
	s, result := readRTFStream(t, readTestdata(t, "word-style.rtf"), Limits{})
	want := []string{
		"Word Heading", "Second level",
		"Plain bold and italic café “quoted” – mono a link. added",
		"Bullet one", "Bullet two1", "Numbered", "Привет При日 😀",
		"omitted:image", " picture, ", "omitted:embedded object", " object, ", "omitted:image",
		"table[Merged across|Merged down|Left|Middle]", "pageBreak:", "New page", "Notes", "1 Footnote text.",
	}
	if got := texts(s.blocks); !reflect.DeepEqual(got, want) {
		t.Fatalf("blocks = %q", got)
	}
	if s.meta.Title != "Word style sample" {
		t.Errorf("title = %q", s.meta.Title)
	}
	if b := s.blocks[1]; b.Type != BlockHeading || b.Level != 2 {
		t.Errorf("an aliased heading style = %+v", b)
	}
	if b := s.blocks[2]; b.Align != "justify" {
		t.Errorf("justified paragraph = %+v", b)
	}
	table := s.blocks[12]
	if c := table.Rows[0].Cells; c[0].ColSpan != 2 || c[1].RowSpan != 2 {
		t.Errorf("spans from the grid and a vertical merge = %+v", table.Rows)
	}
	wantWarnings := []string{"Headers and footers are not shown", "Comments are not shown", "2 images not shown", "1 embedded object not shown"}
	if !reflect.DeepEqual(result.Warnings, wantWarnings) {
		t.Errorf("warnings = %q", result.Warnings)
	}
	for _, secret := range []string{"Page header", "A comment", "hidden", "removed", "A. Writer", "Secret", "template", "HYPERLINK", "PAGE"} {
		for _, b := range texts(s.blocks) {
			if strings.Contains(b, secret) {
				t.Errorf("%q is shown: %q", secret, b)
			}
		}
	}
}

func TestRTFUnicode(t *testing.T) {
	cases := map[string]string{
		`\u8364?`:                     "€",
		`\uc2 \u26085\'93\'fa!`:       "日!",
		`\uc0 \u26085!`:               "日!",
		`\u-10179?\u-8704?`:           "😀",
		`\u-10179?x`:                  "x",
		`\u-8704?`:                    "",
		`{\uc3 \u8364 abc}d`:          "€d",
		`\u8364{?}x`:                  "€?x",
		`\u8364\'80x`:                 "€x",
		`\u8364\b x`:                  "€x",
		`\u0 ?x`:                      "x",
		`\u99999999?`:                 "",
		`\u8226\'95 x`:                "• x",
		"\\u233 ?caf\\'e9\r\n!":       "écafé!",
		`a\~b\_c\-d\emdash\enspace e`: "a b-cd— e",
		`\lquote\rquote\ldblquote\rdblquote\bullet`: "‘’“”•",
		`\\ \{ \}`:          `\ { }`,
		`a\tab b\line c`:    "a\tb\nc",
		"x\x01\x7fy":        "xy",
		`\'0b\'09`:          "\t",
		`\'zz\'4`:           "zz",
		`\unknownword x\*y`: "xy",
		`\verylongcontrolwordthatgoespastthirtytwoletters x`: "x",
		`\par3 x`: "x",
	}
	for body, want := range cases {
		s, _ := readRTFStream(t, rtfDoc(body), Limits{})
		var got []string
		for _, b := range s.blocks {
			got = append(got, plain(b))
		}
		if strings.Join(got, "") != want {
			t.Errorf("%q: got %q, want %q", body, got, want)
		}
	}
}

func TestRTFCodePages(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"Windows-1252", `{\rtf1\ansi \'80\'93\'e9\'81}`, "€“é"},
		{"the document's code page", `{\rtf1\ansi\ansicpg1251 \'cf\'f0\'e8}`, "При"},
		{"a font's charset", rtfDoc(`\f3 \'cf\'f0\'e8 \f0 \'e9`), "При é"},
		{"a font's code page", `{\rtf1{\fonttbl{\f0\cpg1253 Greek;}}\f0 \'e1\'e2}`, "αβ"},
		{"Central European", `{\rtf1\ansicpg1250 \'9a\'e8}`, "šč"},
		{"Macintosh", `{\rtf1\mac \'8e\'87}`, "éá"},
		{"DOS", `{\rtf1\pc \'b3\'82}`, "│é"},
		{"an unknown code page", `{\rtf1\ansicpg9999 \'e9}`, "é"},
		{"a symbol font's bullet", rtfDoc(`\f1 \'b7\'41\'c5`), "•A"},
		{"Shift JIS", `{\rtf1\ansicpg932 a\'82\'a0b\'b1}`, "abｱ"},
		{"a flat font table", `{\rtf1{\fonttbl\f0\fcharset204 Cyr;\f1\fcharset0 Latin;}\f0\'cf\f1\'e9}`, "Пé"},
	}
	for _, c := range cases {
		s, result := readRTFStream(t, c.src, Limits{})
		var got []string
		for _, b := range s.blocks {
			got = append(got, plain(b))
		}
		if strings.Join(got, "") != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
		if c.name == "Shift JIS" && !reflect.DeepEqual(result.Warnings, []string{warnEncoding}) {
			t.Errorf("Shift JIS: warnings = %q", result.Warnings)
		}
	}
}

func TestRTFRunFormatting(t *testing.T) {
	s, _ := readRTFStream(t, rtfDoc(`n {\b b}{\b0 n}{\i i}{\ul u\ulnone n}{\uldb u}{\strike s}{\striked1 s}{\striked0 n}`+
		`{\super p}{\sub d}{\super\nosupersub n}{\up6 p}{\dn4 d}{\up0 n}{\b\i\plain n}{\f2 c}{\v h}{\deleted x}{\b {\i bi}}\par`), Limits{})
	want := []Run{
		{Text: "n "}, {Text: "b", Bold: true}, {Text: "n"}, {Text: "i", Italic: true}, {Text: "u", Underline: true}, {Text: "n"},
		{Text: "u", Underline: true}, {Text: "ss", Strike: true}, {Text: "n"}, {Text: "p", Superscript: true},
		{Text: "d", Subscript: true}, {Text: "n"}, {Text: "p", Superscript: true}, {Text: "d", Subscript: true}, {Text: "nn"},
		{Text: "c", Code: true}, {Text: "bi", Bold: true, Italic: true},
	}
	if len(s.blocks) != 1 || !reflect.DeepEqual(s.blocks[0].Runs, want) {
		t.Fatalf("blocks = %+v", s.blocks)
	}
}

func TestRTFHeadings(t *testing.T) {
	sheet := `{\stylesheet{\s0 Normal;}{\s1\outlinelevel0 Custom;}{\s2 heading 2;}{\s3 Heading 3,H3;}{\s4 Title;}{\cs5 heading 1;}{\*\cs6 heading 1;}{\s7\outlinelevel9 heading 1;}}`
	s, _ := readRTFStream(t, rtfDoc(sheet+`\pard\s1 a\par\pard\s2 b\par\pard\s3 c\par\pard\s4 d\par\pard\s5 e\par\pard\s6 f\par\pard\s7 g\par`+
		`\pard\outlinelevel2 h\par\pard\outlinelevel8 i\par\pard\s2\outlinelevel9 j\par\pard\s9 k\par`), Limits{})
	want := []int{1, 2, 3, 1, 0, 0, 0, 3, 6, 0, 0}
	if len(s.blocks) != len(want) {
		t.Fatalf("blocks = %+v", s.blocks)
	}
	for i, b := range s.blocks {
		if b.Level != want[i] || (want[i] > 0) != (b.Type == BlockHeading) {
			t.Errorf("block %d (%s) = %+v, want heading %d", i, plain(b), b, want[i])
		}
	}
}

func TestRTFLists(t *testing.T) {
	s, _ := readRTFStream(t, rtfDoc(
		`{\listtext\pard\plain\f1 \'b7\tab}\pard\ls1\ilvl0 a\par`+
			`{\listtext\pard\plain 1.\tab}\pard\ls2 b\par`+
			`{\listtext\pard\plain a)\tab}\pard\ls2\ilvl1 c\par`+
			`{\pntext\f1\'b7\tab}{\*\pn\pnlvlblt\pnf1{\pntxtb\'b7}}\pard d\par`+
			`{\listtext o\tab}\pard\ls3\ilvl2 e\par`+
			`\pard\ls4\ilvl12 f\par`+
			`{\listtext\pard - \tab}\pard g\par`+
			`\pard h\par`), Limits{})
	want := []struct {
		typ     BlockType
		level   int
		ordered bool
		marker  string
	}{
		{BlockListItem, 0, false, "•"}, {BlockListItem, 0, true, "1."}, {BlockListItem, 1, true, "a)"},
		{BlockListItem, 0, false, "•"}, {BlockListItem, 2, false, "◦"}, {BlockListItem, 8, false, "•"},
		{BlockListItem, 0, false, "-"}, {BlockParagraph, 0, false, ""},
	}
	if len(s.blocks) != len(want) {
		t.Fatalf("blocks = %+v", s.blocks)
	}
	for i, w := range want {
		b := s.blocks[i]
		if b.Type != w.typ || b.Level != w.level || b.Ordered != w.ordered || b.Marker != w.marker || plain(b) != string(rune('a'+i)) {
			t.Errorf("block %d = %+v, want %+v", i, b, w)
		}
	}
}

func TestRTFTables(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"a plain table", `\trowd\cellx100\cellx200\pard\intbl a\cell b\cell\row\trowd\cellx100\cellx200\pard\intbl c\cell d\cell\row\pard after\par`,
			`[[a(1,1) b(1,1)] [c(1,1) d(1,1)]] after`},
		{"cells spanning the grid", `\trowd\cellx300\cellx400\intbl a\cell b\cell\row\trowd\cellx100\cellx300\cellx400\intbl c\cell d\cell e\cell\row`,
			`[[a(2,1) b(1,1)] [c(1,1) d(1,1) e(1,1)]]`},
		{"an indented row", `\trowd\trleft100\cellx200\intbl a\cell\row\trowd\cellx100\cellx200\intbl b\cell c\cell\row`,
			`[[a(1,1)] [b(1,1) c(1,1)]]`},
		{"merged across", `\trowd\clmgf\cellx100\clmrg\cellx200\cellx300\intbl a\cell x\cell b\cell\row`,
			`[[a/x(2,1) b(1,1)]]`},
		{"merged down", `\trowd\clvmgf\cellx100\cellx200\intbl a\cell b\cell\row\trowd\clvmrg\cellx100\cellx200\intbl x\cell c\cell\row\trowd\clvmrg\cellx100\cellx200\intbl \cell d\cell\row`,
			`[[a/x(1,3) b(1,1)] [c(1,1)] [d(1,1)]]`},
		{"merged across three cells", `\trowd\clmgf\cellx100\clmrg\cellx200\clmrg\cellx300\cellx400\intbl a\cell x\cell y\cell b\cell\row`,
			`[[a/x/y(3,1) b(1,1)]]`},
		{"merged down in the second column", `\trowd\cellx100\clvmgf\cellx200\intbl a\cell b\cell\row\trowd\cellx100\clvmrg\cellx200\intbl c\cell x\cell\row`,
			`[[a(1,1) b/x(1,2)] [c(1,1)]]`},
		{"merged across and down", `\trowd\clmgf\clvmgf\cellx100\clmrg\clvmrg\cellx200\cellx300\intbl a\cell x\cell b\cell\row\trowd\clmgf\clvmrg\cellx100\clmrg\clvmrg\cellx200\cellx300\intbl y\cell z\cell c\cell\row`,
			`[[a/x/y/z(2,2) b(1,1)] [c(1,1)]]`},
		{"a merge that never started", `\trowd\clvmrg\cellx100\intbl a\cell\row`, `[[a(1,1)]]`},
		{"row properties after the cells", `\intbl a\cell b\cell\trowd\clmgf\cellx100\clmrg\cellx200\row`, `[[a/b(2,1)]]`},
		{"more cells than definitions", `\trowd\cellx100\intbl a\cell b\cell\row`, `[[a(1,1) b(1,1)]]`},
		{"several paragraphs in a cell", `\trowd\cellx100\intbl a\par b\cell\row`, `[[a/b(1,1)]]`},
		{"an empty cell", `\trowd\cellx100\cellx200\intbl \cell b\cell\row`, `[[(1,1) b(1,1)]]`},
		{"a table at the end of the file", `\trowd\cellx100\intbl a\cell b`, `[[a(1,1) b(1,1)]]`},
		{"a row without its end", `\trowd\cellx100\intbl a\cell\pard after\par`, `[[a(1,1)]] after`},
		{"a stray row end", `a\par\row b\par`, `a b`},
		{"a nested table", `\trowd\cellx500\pard\intbl outer\par\pard\intbl\itap2 x\nestcell y\nestcell{\*\nesttableprops\trowd\clmgf\cellx100\clmrg\cellx200\nestrow}{\nonesttables\par}\pard\intbl\cell\row`,
			`[[outer/[[x/y(2,1)]](1,1)]]`},
		{"a page break in a table", `\trowd\cellx100\intbl a\page b\cell\row`, `[[ab(1,1)]]`},
		{"a cell end in a note", `a{\footnote\pard\intbl n\cell m\row}\par`, `a Notes n m`},
	}
	for _, c := range cases {
		s, _ := readRTFStream(t, rtfDoc(c.body), Limits{})
		if got := tableShape(s.blocks); got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
}

// tableShape writes blocks compactly: a table as its rows of cells, each
// cell its blocks' text and its column and row span.
func tableShape(blocks []Block) string {
	var parts []string
	for _, b := range blocks {
		if b.Type != BlockTable {
			parts = append(parts, plain(b))
			continue
		}
		var rows []string
		for _, row := range b.Rows {
			var cells []string
			for _, c := range row.Cells {
				var inner []string
				for _, cb := range c.Blocks {
					inner = append(inner, tableShape([]Block{cb}))
				}
				cells = append(cells, strings.Join(inner, "/")+"("+itoa(c.ColSpan)+","+itoa(c.RowSpan)+")")
			}
			rows = append(rows, "["+strings.Join(cells, " ")+"]")
		}
		parts = append(parts, "["+strings.Join(rows, " ")+"]")
	}
	return strings.Join(parts, " ")
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

func TestRTFNestedTablesFlattenPastTheDepthLimit(t *testing.T) {
	body := `\trowd\cellx500\pard\intbl\itap2 deep\nestcell{\*\nesttableprops\trowd\cellx100\nestrow}\pard\intbl\cell\row`
	s, _ := readRTFStream(t, rtfDoc(body), Limits{MaxTableDepth: 1})
	if got := tableShape(s.blocks); got != `[[deep(1,1)]]` {
		t.Errorf("got %s", got)
	}
}

func TestRTFSkippedDestinations(t *testing.T) {
	body := `{\*\unknown text {nested {deeper}} more}{\colortbl;\red1;}{\info{\author Someone}{\title  Two   words }}` +
		`{\*\bkmkstart b}{\xe index entry}{\tc toc entry}{\*\datastore 0102}{\*\themedata 0304}` +
		`{\*\shppict{\pict\pngblip 89504e47}}{\nonshppict{\pict\wmetafile8 0100}}` +
		`{\pict\bin6 }{\}{x}` +
		`{\object{\*\objdata 0102}{\result shown?}}{\shp{\*\shpinst{\sp{\sn x}{\sv y}}}{\shprslt fallback}}` +
		`{\mmath{\*\moMathPara math}}{\header head}{\footerf foot}{\*\annotation comment}{\ftnsep sep}` +
		`{\*\fldinst orphan}visible\par`
	s, result := readRTFStream(t, rtfDoc(body), Limits{})
	want := []string{"omitted:image", "omitted:image", "omitted:embedded object", "omitted:shape", "omitted:equation", "visible"}
	if got := texts(s.blocks); !reflect.DeepEqual(got, want) {
		t.Fatalf("blocks = %q", got)
	}
	if s.meta.Title != "Two words" {
		t.Errorf("title = %q", s.meta.Title)
	}
	wantWarnings := []string{"2 images not shown", "1 embedded object not shown", "1 shape not shown", "1 equation not shown",
		"Headers and footers are not shown", "Comments are not shown"}
	if !reflect.DeepEqual(result.Warnings, wantWarnings) {
		t.Errorf("warnings = %q", result.Warnings)
	}
}

func TestRTFFields(t *testing.T) {
	cases := []struct{ name, field, link string }{
		{"a web link", `HYPERLINK "https://example.com/x y"`, "https://example.com/x y"},
		{"an unquoted link", `HYPERLINK http://example.com`, "http://example.com"},
		{"a mail link", ` hyperlink "mailto:a@example.com" \\o "tip"`, "mailto:a@example.com"},
		{"a tooltip first", `HYPERLINK \\o "tip" "https://example.com"`, "https://example.com"},
		{"a bookmark", `HYPERLINK \\l "_Toc1"`, ""},
		{"a script", `HYPERLINK "javascript:alert(1)"`, ""},
		{"a file", `HYPERLINK "file:///etc/passwd"`, ""},
		{"a remote picture", `INCLUDEPICTURE "https://evil.example/x.png" \\d`, ""},
		{"no instructions", ``, ""},
		{"an unclosed quote", `HYPERLINK "https://example.com`, "https://example.com"},
	}
	for _, c := range cases {
		s, _ := readRTFStream(t, rtfDoc(`{\field{\*\fldinst {`+c.field+`}}{\fldrslt {\ul result}}} after\par`), Limits{})
		want := []Run{{Text: "result", Underline: true, Link: c.link}, {Text: " after"}}
		if len(s.blocks) != 1 || !reflect.DeepEqual(s.blocks[0].Runs, want) {
			t.Errorf("%s: blocks = %+v", c.name, s.blocks)
		}
	}
	s, _ := readRTFStream(t, rtfDoc(`{\field{\*\fldinst HYPERLINK "https://a.example"}{\fldrslt {\field{\*\fldinst PAGE}{\fldrslt 3}} x}}{\field{\*\fldinst DATE}}\par`), Limits{})
	if want := []Run{{Text: "3 x", Link: "https://a.example"}}; len(s.blocks) != 1 || !reflect.DeepEqual(s.blocks[0].Runs, want) {
		t.Errorf("nested fields: blocks = %+v", s.blocks)
	}
}

func TestRTFNotes(t *testing.T) {
	body := `One{\super\chftn}{\footnote\pard\plain{\super\chftn} First note.\par Second paragraph.}` +
		` two{\super\chftn}{\footnote{\super\chftn} Second note{\footnote nested}.}\par` +
		` custom{\footnote * Custom mark.}\par`
	s, result := readRTFStream(t, rtfDoc(body), Limits{})
	want := []string{"One1 two2", "custom", "Notes", "1 First note.", "Second paragraph.", "2 Second note.", "* Custom mark."}
	if got := texts(s.blocks); !reflect.DeepEqual(got, want) {
		t.Fatalf("blocks = %q", got)
	}
	if s.end.Words != 14 || result.Paragraphs != 6 {
		t.Errorf("counts: end %+v, result %+v", s.end, result)
	}

	// Notes past what may be held are left out, and said to be.
	many := strings.Repeat(`x{\footnote note}`, 10) + `\par`
	s, result = readRTFStream(t, `{\rtf1 `+many+`}`, Limits{MaxDefinitions: 3})
	if got := texts(s.blocks); len(got) != 5 || got[1] != "Notes" {
		t.Errorf("blocks = %q", got)
	}
	if !reflect.DeepEqual(result.Warnings, []string{warnNotes}) {
		t.Errorf("warnings = %q", result.Warnings)
	}
}

func TestRTFPageBreaks(t *testing.T) {
	s, _ := readRTFStream(t, rtfDoc(`a\page b\par\pard\pagebb c\par\pard\pagebb\page d\par\pard{\footnote n\page m}\sect e\sect\sect`), Limits{})
	want := []string{"a", "pageBreak:", "b", "pageBreak:", "c", "pageBreak:", "d", "e", "Notes", "nm"}
	if got := texts(s.blocks); !reflect.DeepEqual(got, want) {
		t.Fatalf("blocks = %q", got)
	}
}

func TestRTFEmptyParagraphsAndCounts(t *testing.T) {
	s, result := readRTFStream(t, rtfDoc(`\par one two\par\par three\line four\par`), Limits{})
	if got := texts(s.blocks); !reflect.DeepEqual(got, []string{"", "one two", "", "three\nfour"}) {
		t.Fatalf("blocks = %q", got)
	}
	if result.Words != 4 || result.Paragraphs != 2 || result.Characters != 16 || result.Blocks != 4 {
		t.Errorf("result = %+v", result)
	}
}

func TestRTFReadsWhatAnUnfinishedFileHas(t *testing.T) {
	cases := map[string][]string{
		`{\rtf1 a\par b`:                    {"a", "b"},
		`{\rtf1 a{\b b`:                     {"ab"},
		`{\rtf1 a{\footnote n`:              {"a", "Notes", "n"},
		`{\rtf1 a}trailing text`:            {"a"},
		`{\rtf1 a}}}`:                       {"a"},
		"\xef\xbb\xbf \t{\\rtf1 bom}":       {"bom"},
		`{\rtf1 a\`:                         {"a"},
		`{\rtf1 a\'`:                        {"a"},
		`{\rtf1 a\'4`:                       {"a"},
		`{\rtf1 a\u`:                        {"a"},
		`{\rtf1 a\u-`:                       {"a"},
		`{\rtf1 a\bin`:                      {"a"},
		`{\rtf1 a\bin99 xy`:                 {"a"},
		`{\rtf1 a\b-c}`:                     {"a-c"},
		`{\rtf1 a\b-`:                       {"a-"},
		`{\rtf1 {\*}x}`:                     {"x"},
		`{\rtf1 \u99999999999999999999 ?x}`: {"x"},
		`{\rtf1 \uc99999999999 \u8364 abc}`: {"€"},
		`{\rtf1 \itap99999999999 x}`:        {"table[table[table[table[table[table[table[table[x]]]]]]]]"},
		`{\rtf1 \ilvl-5 \ls1 x}`:            {"x"},
		`{\rtf1 \trowd\cellx-5\cellx-9 \intbl a\cell\row}`: {"table[a]"},
	}
	for src, want := range cases {
		out, _, err := convertRTF(t, src, OutputNDJSON, Limits{})
		if err != nil {
			t.Errorf("%q: %v", src, err)
			continue
		}
		if got := texts(parseStream(t, out).blocks); !reflect.DeepEqual(got, want) {
			t.Errorf("%q: blocks = %q, want %q", src, got, want)
		}
	}
}

func TestRTFMalformed(t *testing.T) {
	for _, src := range []string{"", "plain text", `{\rtfx}`, `{\foo1 x}`, `\rtf1 x`, `{{\rtf1}}`, "<html>{\\rtf1}</html>", strings.Repeat(" ", 20) + `{\rtf1}`} {
		out, _, err := convertRTF(t, src, OutputNDJSON, Limits{})
		if !errors.Is(err, ErrMalformed) {
			t.Errorf("%q: err = %v, want ErrMalformed", src, err)
		}
		if out != "" {
			t.Errorf("%q: wrote %q before failing", src, out)
		}
	}
}

func TestRTFNestingBombFails(t *testing.T) {
	src := `{\rtf1 ` + strings.Repeat("{", 100_000) + strings.Repeat("}", 100_000) + `}`
	var err error
	if n := allocated(func() { _, _, err = convertRTF(t, src, OutputNDJSON, Limits{}) }); n > 4<<20 {
		t.Errorf("a nesting bomb allocated %d bytes", n)
	}
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v, want ErrTooLarge", err)
	}

	// Inside a group being skipped, nesting costs nothing.
	src = rtfDoc(`{\pict ` + strings.Repeat("{", 100_000) + strings.Repeat("}", 100_000) + `}after\par`)
	s, _ := readRTFStream(t, src, Limits{})
	if got := texts(s.blocks); !reflect.DeepEqual(got, []string{"omitted:image", "after"}) {
		t.Errorf("blocks = %q", got)
	}
}

func TestRTFSkippedDataIsNeverHeld(t *testing.T) {
	const size = 32 << 20
	hex := strings.Repeat("0123456789abcdef", size/16)
	binary := strings.Repeat("{}\\", size/3)
	for name, src := range map[string]string{
		"picture hex":   rtfDoc(`{\pict\pngblip ` + hex + `}after\par`),
		"binary data":   rtfDoc(`{\pict\bin` + itoa(len(binary)) + ` ` + binary + `}after\par`),
		"object data":   rtfDoc(`{\object{\*\objdata ` + hex + `}}after\par`),
		"unknown words": rtfDoc(strings.Repeat(`\qqq`, size/4) + ` after\par`),
	} {
		var s stream
		n := allocated(func() {
			out, _, err := convertRTF(t, src, OutputNDJSON, Limits{})
			if err != nil {
				t.Fatal(err)
			}
			s = parseStream(t, out)
		})
		if n > 2<<20 {
			t.Errorf("%s: reading 32 MiB allocated %d bytes", name, n)
		}
		if got := texts(s.blocks); got[len(got)-1] != "after" {
			t.Errorf("%s: blocks = %q", name, got)
		}
	}
}

func TestRTFLimitsTruncate(t *testing.T) {
	s, result := readRTFStream(t, rtfDoc(strings.Repeat(`word\par `, 100)), Limits{MaxBlocks: 10})
	if len(s.blocks) != 10 || !s.end.Truncated || !result.Truncated || result.Blocks != 10 {
		t.Errorf("block limit: %d blocks, end %+v", len(s.blocks), s.end)
	}

	const limit = 64 << 10
	out, result, err := convertRTF(t, rtfDoc(strings.Repeat("x", 1<<20)+`\par after\par`), OutputNDJSON, Limits{MaxOutputBytes: limit})
	if err != nil {
		t.Fatal(err)
	}
	s = parseStream(t, out)
	if !result.Truncated || len(s.blocks) != 1 || len(out) > limit+1024 || len(plain(s.blocks[0])) < limit/8 {
		t.Errorf("output limit: truncated %v, %d blocks, %d bytes", result.Truncated, len(s.blocks), len(out))
	}

	row := `\trowd\cellx100\cellx200\intbl a cell with some words\cell another\cell\row`
	for _, output := range []Output{OutputNDJSON, OutputDelta} {
		out, result, err := convertRTF(t, rtfDoc(`before\par`+strings.Repeat(row, 1000)+`\pard after\par`), output, Limits{MaxOutputBytes: 8 << 10})
		if err != nil {
			t.Fatal(err)
		}
		if !result.Truncated || len(out) > 16<<10 {
			t.Fatalf("output %d: truncated %v, %d bytes", output, result.Truncated, len(out))
		}
		if output != OutputNDJSON {
			continue
		}
		s := parseStream(t, out)
		if len(s.blocks) != 2 || s.blocks[1].Type != BlockTable {
			t.Fatalf("blocks = %q", texts(s.blocks))
		}
		if rows := len(s.blocks[1].Rows); rows == 0 || rows >= 1000 {
			t.Errorf("the truncated table has %d rows", rows)
		}
	}

	// A limit reached inside a note keeps the body paragraph it is in.
	s, result = readRTFStream(t, rtfDoc(`kept{\footnote `+strings.Repeat(`n\par `, 50)+`}\par`), Limits{MaxBlocks: 5})
	if got := texts(s.blocks); !result.Truncated || len(got) != 1 || got[0] != "kept" {
		t.Errorf("a limit in a note: blocks %q, result %+v", got, result)
	}
}

func TestRTFTablesAndStylesAreBounded(t *testing.T) {
	fonts := `{\rtf1{\fonttbl` + strings.Repeat(`{\f1 F;}{\f2 G;}`, 2) + `{\f3 H;}}x}`
	if _, _, err := convertRTF(t, fonts, OutputNDJSON, Limits{MaxDefinitions: 2}); !errors.Is(err, ErrTooLarge) {
		t.Errorf("fonts: err = %v, want ErrTooLarge", err)
	}
	styles := `{\rtf1{\stylesheet` + strings.Repeat(`{\s1 heading 1;}`, 3) + `}x}`
	if _, _, err := convertRTF(t, styles, OutputNDJSON, Limits{MaxDefinitions: 2}); !errors.Is(err, ErrTooLarge) {
		t.Errorf("styles: err = %v, want ErrTooLarge", err)
	}
	long := `{\rtf1{\fonttbl{\f0 ` + strings.Repeat("Mono", 10_000) + `;}}{\stylesheet{\s1 ` + strings.Repeat("x", 10_000) + `;}}` +
		`{\info{\title ` + strings.Repeat("t", 10_000) + `}}{\field{\*\fldinst HYPERLINK "https://` + strings.Repeat("a", 10_000) + `"}{\fldrslt x}}}`
	n := allocated(func() {
		s, _ := readRTFStream(t, long, Limits{})
		if len(s.meta.Title) > maxRTFTitleBytes || len(s.blocks) != 1 || s.blocks[0].Runs[0].Link != "" || !s.blocks[0].Runs[0].Code {
			t.Errorf("stream = %+v", s)
		}
	})
	if n > 1<<20 {
		t.Errorf("long names allocated %d bytes", n)
	}
	cells := `{\rtf1\trowd` + strings.Repeat(`\cellx100`, 1000) + `\intbl a\cell\row}`
	s, _ := readRTFStream(t, cells, Limits{})
	if got := tableShape(s.blocks); got != `[[a(1,1)]]` {
		t.Errorf("a row of 1000 cell definitions: %s", got)
	}
}

func TestRTFContextEndsTheConversion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, src := range []string{rtfDoc(`x\par`), rtfDoc(`{\pict ` + strings.Repeat("00", 1<<16) + `}`)} {
		e := newEmitter(ConvertParams{Ctx: ctx, Format: FormatRTF, Out: io.Discard})
		if err := readRTF(e, strings.NewReader(src), int64(len(src))); !errors.Is(err, context.Canceled) {
			t.Errorf("err = %v, want context.Canceled", err)
		}
	}
}

func TestRTFWriteErrorsAreReturnedAsTheyAre(t *testing.T) {
	src := rtfDoc(`x\par`)
	for _, output := range []Output{OutputNDJSON, OutputDelta} {
		e := newEmitter(ConvertParams{Format: FormatRTF, Output: output, Out: failingWriter{}})
		if err := readRTF(e, strings.NewReader(src), int64(len(src))); !errors.Is(err, errWrite) {
			t.Errorf("output %d: err = %v, want the write error", output, err)
		}
	}
}

func TestRTFDelta(t *testing.T) {
	out, result, err := convertRTF(t, readTestdata(t, "word-style.rtf"), OutputDelta, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	var doc deltaDoc
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("not a Delta: %v\n%s", err, out)
	}
	if !strings.Contains(out, `"insert":"Word Heading"`) || !strings.Contains(out, `"header":1`) || !strings.Contains(out, `"list":"bullet"`) {
		t.Errorf("Delta = %s", out)
	}
	if !reflect.DeepEqual(result.Warnings[len(result.Warnings)-1:], []string{"1 table flattened to text"}) {
		t.Errorf("warnings = %q", result.Warnings)
	}
}

// fuzzRTFSeeds are RTF snippets that reach every part of the reader.
var fuzzRTFSeeds = []string{
	rtfDoc(`{\stylesheet{\s1 heading 1;}}\pard\s1 H\par{\listtext 1.\tab}\pard\ls1 i\par{\b\i\ul\strike\super x}\par`),
	rtfDoc(`\uc2\u26085\'93\'fa\u-10179??\u-8704??\f3\'cf\f1\'b7\par`),
	rtfDoc(`\trowd\clvmgf\clmgf\cellx100\clmrg\cellx200\intbl a\cell b\cell\row\trowd\clvmrg\cellx100\cellx200\intbl\itap2 c\nestcell{\*\nesttableprops\trowd\cellx50\nestrow}\pard\intbl\cell d\cell\row`),
	rtfDoc(`{\field{\*\fldinst HYPERLINK "https://example.com"}{\fldrslt x}}{\pict\bin3 {}}{\object x}{\footnote{\super\chftn} n}\page\pard\pagebb y`),
	`{\rtf1\ansicpg932 \'82\'a0\'b1{\info{\title t}}{\*\x{\y}}{\header h}}`,
}

// FuzzRTF reads arbitrary bytes as an RTF file, without Convert's recover,
// so a panic fails the target, and checks that what it writes is well
// formed.
func FuzzRTF(f *testing.F) {
	for _, seed := range fuzzRTFSeeds {
		f.Add([]byte(seed))
	}
	entries, err := os.ReadDir(filepath.Join("testdata", "rtf"))
	if err != nil {
		f.Fatal(err)
	}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join("testdata", "rtf", entry.Name()))
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		for _, output := range []Output{OutputNDJSON, OutputDelta} {
			var out bytes.Buffer
			e := newEmitter(ConvertParams{Format: FormatRTF, Output: output, Out: &out, Limits: fuzzLimits})
			err := readRTF(e, bytes.NewReader(data), int64(len(data)))
			if err != nil && !errors.Is(err, errTruncated) {
				if !errors.Is(err, ErrMalformed) && !errors.Is(err, ErrTooLarge) {
					t.Fatalf("an error that is neither malformed nor too large: %v", err)
				}
				continue
			}
			if err := e.finish(); err != nil {
				t.Fatal(err)
			}
			if output == OutputDelta {
				var doc deltaDoc
				if err := json.Unmarshal(out.Bytes(), &doc); err != nil || len(doc.Ops) == 0 {
					t.Fatalf("the Delta is not a document: %v\n%s", err, out.String())
				}
				continue
			}
			lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
			for i, line := range lines {
				if !json.Valid([]byte(line)) {
					t.Fatalf("line %d is not JSON: %s", i, line)
				}
			}
			if !strings.HasPrefix(lines[0], `{"type":"meta"`) || !strings.HasPrefix(lines[len(lines)-1], `{"type":"end"`) {
				t.Fatalf("the stream does not run from meta to end:\n%s", out.String())
			}
		}
	})
}
