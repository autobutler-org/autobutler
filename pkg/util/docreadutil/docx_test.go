package docreadutil

// cspell:ignore notbold notbolda MCMXCIV movedaway MERGEFORMAT scriptfileanchor descr txbx abstractNum abstractNumId altChunk basedOn blip delText docPr endnote endnoteRef endnoteReference fldChar fldCharType fldSimple footnoteRef footnoteReference graphicData gridSpan ilvl imagedata instrText jc lvl lvlOverride lvlText moveFrom numFmt numId numPr oMath outlineLvl pPr pStyle rFonts rPr rStyle sdt sdtContent startOverride styleId tbl tc tcPr tr trPr vbaProject vertAlign vMerge Consolas

import (
	"reflect"
	"strings"
	"testing"
)

const testStyles = `
<w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="Calibri"/></w:rPr></w:rPrDefault></w:docDefaults>
<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/></w:style>
<w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:basedOn w:val="Normal"/><w:pPr><w:outlineLvl w:val="0"/></w:pPr><w:rPr><w:b/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="heading 2"/><w:basedOn w:val="Heading1"/></w:style>
<w:style w:type="paragraph" w:styleId="MyOutline"><w:name w:val="Custom"/><w:pPr><w:outlineLvl w:val="2"/></w:pPr></w:style>
<w:style w:type="paragraph" w:styleId="Title"><w:name w:val="Title"/></w:style>
<w:style w:type="paragraph" w:styleId="Centered"><w:name w:val="Centered"/><w:pPr><w:jc w:val="center"/></w:pPr></w:style>
<w:style w:type="paragraph" w:styleId="ListBullet"><w:name w:val="List Bullet"/><w:pPr><w:numPr><w:numId w:val="2"/></w:numPr></w:pPr></w:style>
<w:style w:type="paragraph" w:styleId="Code"><w:name w:val="Code"/><w:rPr><w:rFonts w:ascii="Consolas"/></w:rPr></w:style>
<w:style w:type="character" w:styleId="Strong"><w:name w:val="Strong"/><w:rPr><w:b/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="LoopA"><w:name w:val="Loop A"/><w:basedOn w:val="LoopB"/></w:style>
<w:style w:type="paragraph" w:styleId="LoopB"><w:name w:val="Loop B"/><w:basedOn w:val="LoopA"/></w:style>
`

const testNumbering = `
<w:abstractNum w:abstractNumId="10">
  <w:lvl w:ilvl="0"><w:start w:val="1"/><w:numFmt w:val="decimal"/><w:lvlText w:val="%1."/></w:lvl>
  <w:lvl w:ilvl="1"><w:start w:val="1"/><w:numFmt w:val="lowerLetter"/><w:lvlText w:val="%1.%2)"/></w:lvl>
  <w:lvl w:ilvl="2"><w:start w:val="1"/><w:numFmt w:val="upperRoman"/><w:lvlText w:val="%3"/></w:lvl>
</w:abstractNum>
<w:abstractNum w:abstractNumId="20">
  <w:lvl w:ilvl="0"><w:numFmt w:val="bullet"/><w:lvlText w:val="` + "\uF0B7" + `"/></w:lvl>
  <w:lvl w:ilvl="1"><w:numFmt w:val="bullet"/><w:lvlText w:val="o"/></w:lvl>
</w:abstractNum>
<w:num w:numId="1"><w:abstractNumId w:val="10"/></w:num>
<w:num w:numId="2"><w:abstractNumId w:val="20"/></w:num>
<w:num w:numId="3"><w:abstractNumId w:val="10"/><w:lvlOverride w:ilvl="0"><w:startOverride w:val="5"/></w:lvlOverride></w:num>
`

// styled is a paragraph in a style.
func styled(style, text string) string {
	return `<w:p><w:pPr><w:pStyle w:val="` + style + `"/></w:pPr><w:r><w:t>` + text + `</w:t></w:r></w:p>`
}

// listed is a paragraph in a list at a level.
func listed(numID, ilvl, text string) string {
	return `<w:p><w:pPr><w:numPr><w:ilvl w:val="` + ilvl + `"/><w:numId w:val="` + numID + `"/></w:numPr></w:pPr><w:r><w:t>` + text + `</w:t></w:r></w:p>`
}

func TestHeadingsAndStyles(t *testing.T) {
	s, _ := readStream(t, testDoc{styles: testStyles, title: "Q3 report", body: styled("Title", "Report") +
		styled("Heading1", "One") + styled("Heading2", "Two") + styled("MyOutline", "Three") +
		`<w:p><w:pPr><w:outlineLvl w:val="7"/></w:pPr><w:r><w:t>Deep</w:t></w:r></w:p>` +
		`<w:p><w:pPr><w:pStyle w:val="Heading1"/><w:outlineLvl w:val="9"/></w:pPr><w:r><w:t>Body</w:t></w:r></w:p>` +
		styled("Centered", "Middle") + styled("LoopA", "Loop") + styled("Missing", "Unknown")}, Limits{})
	if s.meta.Title != "Q3 report" || s.meta.Format != FormatDOCX || s.meta.Version != StreamVersion {
		t.Errorf("meta = %+v", s.meta)
	}
	want := []struct {
		typ   BlockType
		level int
		text  string
	}{
		{BlockHeading, 1, "Report"}, {BlockHeading, 1, "One"}, {BlockHeading, 2, "Two"}, {BlockHeading, 3, "Three"},
		{BlockHeading, 6, "Deep"}, {BlockParagraph, 0, "Body"}, {BlockParagraph, 0, "Middle"},
		{BlockParagraph, 0, "Loop"}, {BlockParagraph, 0, "Unknown"},
	}
	if len(s.blocks) != len(want) {
		t.Fatalf("blocks = %v", texts(s.blocks))
	}
	for i, w := range want {
		if b := s.blocks[i]; b.Type != w.typ || b.Level != w.level || plain(b) != w.text {
			t.Errorf("block %d = %s %d %q, want %s %d %q", i, b.Type, b.Level, plain(b), w.typ, w.level, w.text)
		}
	}
	if !s.blocks[1].Runs[0].Bold {
		t.Error("Heading1's bold from its style is lost")
	}
	if !s.blocks[2].Runs[0].Bold {
		t.Error("Heading2 does not inherit bold from Heading1")
	}
	if s.blocks[6].Align != "center" {
		t.Errorf("Centered align = %q", s.blocks[6].Align)
	}
}

func TestRunFormatting(t *testing.T) {
	body := `<w:p><w:pPr><w:jc w:val="both"/></w:pPr>` +
		`<w:r><w:t xml:space="preserve">Revenue </w:t></w:r>` +
		`<w:r><w:rPr><w:b/></w:rPr><w:t>ro</w:t></w:r><w:r><w:rPr><w:b w:val="1"/></w:rPr><w:t>se</w:t></w:r>` +
		`<w:r><w:rPr><w:i/><w:u w:val="single"/><w:strike/></w:rPr><w:t>x</w:t></w:r>` +
		`<w:r><w:rPr><w:u w:val="none"/><w:vertAlign w:val="superscript"/></w:rPr><w:t>2</w:t></w:r>` +
		`<w:r><w:rPr><w:vertAlign w:val="subscript"/></w:rPr><w:t>n</w:t></w:r>` +
		`<w:r><w:rPr><w:rFonts w:ascii="Courier New"/></w:rPr><w:t>code</w:t></w:r>` +
		`<w:r><w:rPr><w:rStyle w:val="Strong"/></w:rPr><w:t>strong</w:t></w:r>` +
		`<w:r><w:rPr><w:rStyle w:val="Strong"/><w:b w:val="0"/></w:rPr><w:t>notbold</w:t></w:r>` +
		`<w:r><w:rPr><w:vanish/></w:rPr><w:t>hidden</w:t></w:r>` +
		`<w:r><w:t>a</w:t><w:tab/><w:t>b</w:t><w:br/><w:t>c</w:t><w:noBreakHyphen/><w:sym w:font="Symbol" w:char="F0B7"/><w:sym w:char="2014"/></w:r>` +
		`</w:p>` + styled("Code", "mono")
	s, _ := readStream(t, testDoc{styles: testStyles, body: body}, Limits{})
	if len(s.blocks) != 2 {
		t.Fatalf("blocks = %v", texts(s.blocks))
	}
	p := s.blocks[0]
	if p.Align != "justify" {
		t.Errorf("align = %q", p.Align)
	}
	want := []Run{
		{Text: "Revenue "}, {Text: "rose", Bold: true}, {Text: "x", Italic: true, Underline: true, Strike: true},
		{Text: "2", Superscript: true}, {Text: "n", Subscript: true}, {Text: "code", Code: true},
		{Text: "strong", Bold: true}, {Text: "notbolda\tb\nc-—"},
	}
	if !reflect.DeepEqual(p.Runs, want) {
		t.Errorf("runs =\n%+v\nwant\n%+v", p.Runs, want)
	}
	if r := s.blocks[1].Runs; len(r) != 1 || !r[0].Code {
		t.Errorf("a paragraph in a monospace style: runs = %+v", r)
	}
}

func TestLists(t *testing.T) {
	body := listed("1", "0", "one") + listed("1", "1", "one-a") + listed("1", "1", "one-b") +
		listed("1", "2", "deep") + listed("1", "0", "two") + listed("1", "1", "two-a") +
		listed("2", "0", "dot") + listed("2", "1", "circle") + styled("ListBullet", "styled") +
		listed("3", "0", "five") + listed("9", "0", "undefined") + listed("0", "0", "none")
	s, _ := readStream(t, testDoc{styles: testStyles, numbering: testNumbering, body: body}, Limits{})
	want := []struct {
		ordered bool
		level   int
		marker  string
	}{
		{true, 0, "1."}, {true, 1, "1.a)"}, {true, 1, "1.b)"}, {true, 2, "I"}, {true, 0, "2."}, {true, 1, "2.a)"},
		{false, 0, "•"}, {false, 1, "◦"}, {false, 0, "•"}, {true, 0, "5."},
	}
	if len(s.blocks) != len(want)+2 {
		t.Fatalf("blocks = %v", texts(s.blocks))
	}
	for i, w := range want {
		b := s.blocks[i]
		if b.Type != BlockListItem || b.Ordered != w.ordered || b.Level != w.level || b.Marker != w.marker {
			t.Errorf("item %d %q = %s ordered=%v level=%d marker=%q, want %+v", i, plain(b), b.Type, b.Ordered, b.Level, b.Marker, w)
		}
	}
	for _, b := range s.blocks[len(want):] {
		if b.Type != BlockParagraph {
			t.Errorf("%q in an undefined or removed list is a %s, want a paragraph", plain(b), b.Type)
		}
	}
}

func TestNumberFormats(t *testing.T) {
	cases := []struct {
		n      int
		format string
		want   string
	}{
		{3, "decimal", "3"}, {7, "decimalZero", "07"}, {1, "lowerLetter", "a"}, {27, "lowerLetter", "aa"},
		{28, "upperLetter", "BB"}, {4, "lowerRoman", "iv"}, {1994, "upperRoman", "MCMXCIV"}, {4000, "upperRoman", "4000"},
		{0, "lowerLetter", "0"}, {1, "ordinal", "1st"}, {12, "ordinal", "12th"}, {23, "ordinal", "23rd"}, {5, "chicago", "5"},
	}
	for _, c := range cases {
		if got := formatNumber(c.n, c.format); got != c.want {
			t.Errorf("formatNumber(%d, %s) = %q, want %q", c.n, c.format, got, c.want)
		}
	}
}

// cell is a table cell holding one paragraph, with cell properties.
func cell(props, text string) string {
	return `<w:tc><w:tcPr>` + props + `</w:tcPr>` + para(text) + `</w:tc>`
}

func TestTablesWithSpans(t *testing.T) {
	body := `<w:tbl><w:tblPr/><w:tblGrid><w:gridCol/><w:gridCol/><w:gridCol/></w:tblGrid>` +
		`<w:tr>` + cell(`<w:gridSpan w:val="2"/>`, "Header") + cell("", "C") + `</w:tr>` +
		`<w:tr>` + cell(`<w:vMerge w:val="restart"/>`, "Tall") + cell("", "B2") + cell("", "C2") + `</w:tr>` +
		`<w:tr>` + cell(`<w:vMerge/>`, "") + cell("", "B3") + cell("", "C3") + `</w:tr>` +
		`<w:tr><w:trPr><w:del w:id="9" w:author="x"/></w:trPr>` + cell("", "Deleted") + `</w:tr>` +
		`<w:tr>` + cell(`<w:vMerge w:val="continue"/>`, "more") + cell(`<w:gridSpan w:val="2"/>`, "Wide") + `</w:tr>` +
		`</w:tbl>`
	s, _ := readStream(t, testDoc{body: body}, Limits{})
	if len(s.blocks) != 1 || s.blocks[0].Type != BlockTable {
		t.Fatalf("blocks = %v", texts(s.blocks))
	}
	rows := s.blocks[0].Rows
	if len(rows) != 4 {
		t.Fatalf("rows = %d, want 4 (the deleted row dropped)", len(rows))
	}
	shape := func(row TableRow) []string {
		var out []string
		for _, c := range row.Cells {
			out = append(out, strings.Join(texts(c.Blocks), "/")+":"+string(rune('0'+c.ColSpan))+"x"+string(rune('0'+c.RowSpan)))
		}
		return out
	}
	want := [][]string{
		{"Header:2x1", "C:1x1"},
		{"Tall/more:1x3", "B2:1x1", "C2:1x1"},
		{"B3:1x1", "C3:1x1"},
		{"Wide:2x1"},
	}
	for i := range want {
		if got := shape(rows[i]); !reflect.DeepEqual(got, want[i]) {
			t.Errorf("row %d = %v, want %v", i, got, want[i])
		}
	}
}

// nested is a table nesting depth tables, the innermost holding text.
func nested(depth int, text string) string {
	if depth == 0 {
		return para(text)
	}
	return `<w:tbl><w:tr><w:tc>` + nested(depth-1, text) + `</w:tc></w:tr></w:tbl>`
}

func TestNestedTablesFlattenPastTheDepthLimit(t *testing.T) {
	s, _ := readStream(t, testDoc{body: nested(5, "inside")}, Limits{MaxTableDepth: 3})
	depth := 0
	blocks := s.blocks
	for len(blocks) == 1 && blocks[0].Type == BlockTable {
		depth++
		blocks = blocks[0].Rows[0].Cells[0].Blocks
	}
	if depth != 3 || len(blocks) != 1 || plain(blocks[0]) != "inside" {
		t.Fatalf("nesting = %d around %v, want 3 tables around the text", depth, texts(blocks))
	}
}

func TestTrackedChangesReadAsAccepted(t *testing.T) {
	body := `<w:p><w:r><w:t xml:space="preserve">kept </w:t></w:r>` +
		`<w:ins w:id="1" w:author="a"><w:r><w:t xml:space="preserve">inserted </w:t></w:r></w:ins>` +
		`<w:del w:id="2" w:author="a"><w:r><w:delText>deleted </w:delText></w:r></w:del>` +
		`<w:moveFrom w:id="3"><w:r><w:t>movedaway </w:t></w:r></w:moveFrom>` +
		`<w:moveTo w:id="4"><w:r><w:t>moved</w:t></w:r></w:moveTo>` +
		`<w:r><w:rPr><w:b/><w:rPrChange w:id="5"><w:rPr><w:i/></w:rPr></w:rPrChange></w:rPr><w:t>!</w:t></w:r></w:p>` +
		`<w:del w:id="6"><w:p><w:r><w:t>whole paragraph</w:t></w:r></w:p></w:del>`
	s, _ := readStream(t, testDoc{body: body}, Limits{})
	if got := texts(s.blocks); !reflect.DeepEqual(got, []string{"kept inserted moved!"}) {
		t.Fatalf("texts = %q", got)
	}
	if r := s.blocks[0].Runs; r[len(r)-1].Italic || !r[len(r)-1].Bold {
		t.Errorf("a tracked formatting change shows the old formatting: %+v", r[len(r)-1])
	}
}

func TestFieldsShowOnlyTheirResult(t *testing.T) {
	body := `<w:p><w:r><w:t xml:space="preserve">Page </w:t></w:r>` +
		`<w:r><w:fldChar w:fldCharType="begin"/></w:r><w:r><w:instrText>PAGE \* MERGEFORMAT</w:instrText></w:r>` +
		`<w:r><w:t>hidden instruction</w:t></w:r>` +
		`<w:r><w:fldChar w:fldCharType="begin"/></w:r><w:r><w:instrText>NESTED</w:instrText></w:r><w:r><w:fldChar w:fldCharType="separate"/></w:r><w:r><w:t>nested result in instruction</w:t></w:r><w:r><w:fldChar w:fldCharType="end"/></w:r>` +
		`<w:r><w:fldChar w:fldCharType="separate"/></w:r><w:r><w:t>3</w:t></w:r><w:r><w:fldChar w:fldCharType="end"/></w:r>` +
		`<w:fldSimple w:instr="DDEAUTO c:\\windows\\system32\\cmd.exe"><w:r><w:t xml:space="preserve"> of 9</w:t></w:r></w:fldSimple>` +
		`<w:r><w:fldChar w:fldCharType="end"/></w:r><w:r><w:t>.</w:t></w:r></w:p>`
	s, _ := readStream(t, testDoc{body: body}, Limits{})
	if got := texts(s.blocks); !reflect.DeepEqual(got, []string{"Page 3 of 9."}) {
		t.Fatalf("texts = %q", got)
	}
}

func TestHyperlinks(t *testing.T) {
	rels := `<Relationship Id="rLink" Type="` + relNS + `/hyperlink" Target="https://example.com/a?b=1" TargetMode="External"/>` +
		`<Relationship Id="rMail" Type="` + relNS + `/hyperlink" Target="mailto:a@example.com" TargetMode="External"/>` +
		`<Relationship Id="rJS" Type="` + relNS + `/hyperlink" Target="javascript:alert(1)" TargetMode="External"/>` +
		`<Relationship Id="rFile" Type="` + relNS + `/hyperlink" Target="file:///etc/passwd" TargetMode="External"/>`
	body := `<w:p><w:hyperlink r:id="rLink"><w:r><w:t>web</w:t></w:r></w:hyperlink>` +
		`<w:hyperlink r:id="rMail"><w:r><w:t>mail</w:t></w:r></w:hyperlink>` +
		`<w:hyperlink r:id="rJS"><w:r><w:t>script</w:t></w:r></w:hyperlink>` +
		`<w:hyperlink r:id="rFile"><w:r><w:t>file</w:t></w:r></w:hyperlink>` +
		`<w:hyperlink w:anchor="_Toc1"><w:r><w:t>anchor</w:t></w:r></w:hyperlink></w:p>`
	s, _ := readStream(t, testDoc{body: body, rels: rels}, Limits{})
	want := []Run{
		{Text: "web", Link: "https://example.com/a?b=1"}, {Text: "mail", Link: "mailto:a@example.com"},
		{Text: "scriptfileanchor"},
	}
	if got := s.blocks[0].Runs; !reflect.DeepEqual(got, want) {
		t.Fatalf("runs = %+v, want %+v", got, want)
	}
}

// drawing is an inline picture whose blip uses the attribute given.
func drawing(blipAttr, alt string) string {
	return `<w:r><w:drawing><wp:inline><wp:extent cx="6096000" cy="3429000"/><wp:docPr id="1" name="Picture 1" descr="` + alt + `"/>` +
		`<a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:pic><pic:blipFill>` +
		`<a:blip ` + blipAttr + `/></pic:blipFill></pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r>`
}

func TestImagesAndOmittedObjects(t *testing.T) {
	rels := `<Relationship Id="rImg" Type="` + relNS + `/image" Target="media/image1.png"/>` +
		`<Relationship Id="rGone" Type="` + relNS + `/image" Target="media/missing.png"/>` +
		`<Relationship Id="rRemote" Type="` + relNS + `/image" Target="https://tracker.example/pixel.png" TargetMode="External"/>` +
		`<Relationship Id="rVml" Type="` + relNS + `/image" Target="media/image2.jpeg"/>`
	chart := `<w:r><w:drawing><wp:inline><wp:docPr id="2" name="Chart"/><a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/chart"><c:chart xmlns:c="http://schemas.openxmlformats.org/drawingml/2006/chart" r:id="rChart"/></a:graphicData></a:graphic></wp:inline></w:drawing></w:r>`
	textBox := `<w:r><mc:AlternateContent><mc:Choice Requires="wps"><w:drawing><wp:anchor><wp:docPr id="3" name="Text Box"/><a:graphic><a:graphicData uri="http://schemas.microsoft.com/office/word/2010/wordprocessingShape"><wps:wsp xmlns:wps="http://schemas.microsoft.com/office/word/2010/wordprocessingShape"><wps:txbx><w:txbxContent>` + para("boxed") + `</w:txbxContent></wps:txbx></wps:wsp></a:graphicData></a:graphic></wp:anchor></w:drawing></mc:Choice><mc:Fallback><w:pict><v:shape><v:textbox/></v:shape></w:pict></mc:Fallback></mc:AlternateContent></w:r>`
	vml := `<w:r><w:pict><v:shape style="width:72pt;height:36pt" alt="vml alt"><v:imagedata r:id="rVml" o:title=""/></v:shape></w:pict></w:r>`
	object := `<w:r><w:object><v:shape><v:imagedata r:id="rVml"/></v:shape><o:OLEObject Type="Embed" ProgID="Package" r:id="rOle"/></w:object></w:r>`
	body := `<w:p><w:r><w:t>before</w:t></w:r>` + drawing(`r:embed="rImg"`, "Chart of revenue") + `<w:r><w:t>after</w:t></w:r></w:p>` +
		`<w:p>` + drawing(`r:embed="rGone"`, "") + drawing(`r:link="rRemote"`, "Logo") + drawing(`r:embed="rRemote"`, "") + drawing(`r:embed="rNone"`, "") + `</w:p>` +
		`<w:p>` + chart + textBox + vml + object + `<m:oMath><m:r><m:t>x</m:t></m:r></m:oMath></w:p>` +
		`<w:altChunk r:id="rChunk"/>`
	s, result := readStream(t, testDoc{body: body, rels: rels, files: map[string]string{"word/media/image1.png": "png", "word/media/image2.jpeg": "jpeg"}}, Limits{})
	want := []string{
		"before", "image:word/media/image1.png", "after",
		"omitted:image", "image:", "image:", "omitted:image",
		"omitted:chart", "omitted:text box", "image:word/media/image2.jpeg", "omitted:embedded object", "omitted:equation",
		"omitted:embedded document",
	}
	if got := texts(s.blocks); !reflect.DeepEqual(got, want) {
		t.Fatalf("texts =\n%q\nwant\n%q", got, want)
	}
	if img := s.blocks[1]; img.Alt != "Chart of revenue" || img.Width != 640 || img.Height != 360 || img.External {
		t.Errorf("image = %+v", img)
	}
	for _, i := range []int{4, 5} {
		if img := s.blocks[i]; !img.External || img.Part != "" {
			t.Errorf("a linked image must be external with no part: %+v", img)
		}
	}
	if s.blocks[4].Alt != "Logo" {
		t.Errorf("linked image alt = %q", s.blocks[4].Alt)
	}
	if vml := s.blocks[9]; vml.Width != 96 || vml.Height != 48 || vml.Alt != "vml alt" {
		t.Errorf("VML image = %+v", vml)
	}
	wantWarnings := []string{"2 images not shown", "2 linked images not shown", "1 chart not shown", "1 text box not shown",
		"1 embedded object not shown", "1 equation not shown", "1 embedded document not shown"}
	if !reflect.DeepEqual(s.end.Warnings, wantWarnings) || !reflect.DeepEqual(result.Warnings, wantWarnings) {
		t.Errorf("warnings = %q / %q, want %q", s.end.Warnings, result.Warnings, wantWarnings)
	}
	if s.end.Words != 2 {
		t.Errorf("words = %d: text inside an omitted text box or equation is counted", s.end.Words)
	}
}

func TestNotesFollowTheBody(t *testing.T) {
	footnotes := `<w:footnote w:type="separator" w:id="-1"><w:p><w:r><w:separator/></w:r></w:p></w:footnote>` +
		`<w:footnote w:id="1"><w:p><w:r><w:footnoteRef/></w:r><w:r><w:t xml:space="preserve"> First note.</w:t></w:r></w:p></w:footnote>` +
		`<w:footnote w:id="2"><w:p><w:r><w:t>Never referred to.</w:t></w:r></w:p></w:footnote>` +
		`<w:footnote w:id="3"><w:p><w:r><w:footnoteRef/></w:r><w:r><w:t xml:space="preserve"> Second note.</w:t></w:r></w:p></w:footnote>`
	endnotes := `<w:endnote w:id="1"><w:p><w:r><w:endnoteRef/></w:r><w:r><w:t xml:space="preserve"> An endnote.</w:t></w:r></w:p></w:endnote>`
	body := `<w:p><w:r><w:t>Claim</w:t></w:r><w:r><w:footnoteReference w:id="3"/></w:r><w:r><w:t xml:space="preserve"> and </w:t></w:r>` +
		`<w:r><w:footnoteReference w:id="1"/></w:r><w:r><w:endnoteReference w:id="1"/></w:r></w:p>`
	s, _ := readStream(t, testDoc{body: body, footnotes: footnotes, endnotes: endnotes}, Limits{})
	want := []string{"Claim1 and 2i", "Notes", "2 First note.", "1 Second note.", "i An endnote."}
	if got := texts(s.blocks); !reflect.DeepEqual(got, want) {
		t.Fatalf("texts = %q, want %q", got, want)
	}
	if r := s.blocks[0].Runs[1]; !r.Superscript || r.Text != "1" {
		t.Errorf("note mark = %+v, want a superscript 1", r)
	}
	if h := s.blocks[1]; h.Type != BlockHeading || h.Level != 1 {
		t.Errorf("Notes heading = %+v", h)
	}
	if s.end.Words != 3+3+3+3 {
		t.Errorf("words = %d; the Notes heading must not count", s.end.Words)
	}
}

func TestPageBreaks(t *testing.T) {
	body := `<w:p><w:r><w:t>one</w:t><w:br w:type="page"/><w:t>two</w:t></w:r></w:p>` +
		`<w:p><w:pPr><w:pageBreakBefore/></w:pPr><w:r><w:t>three</w:t></w:r></w:p>` +
		`<w:tbl><w:tr><w:tc><w:p><w:r><w:t>a</w:t><w:br w:type="page"/><w:t>b</w:t></w:r></w:p></w:tc></w:tr></w:tbl>`
	s, _ := readStream(t, testDoc{body: body}, Limits{})
	want := []string{"one", "pageBreak:", "two", "pageBreak:", "three", "table[ab]"}
	if got := texts(s.blocks); !reflect.DeepEqual(got, want) {
		t.Fatalf("texts = %q, want %q", got, want)
	}
}

func TestContentControlsAndAlternateContent(t *testing.T) {
	body := `<w:sdt><w:sdtPr><w:alias w:val="secret"/></w:sdtPr><w:sdtContent>` + para("in control") + `</w:sdtContent></w:sdt>` +
		`<w:p><w:sdt><w:sdtContent><w:r><w:t>inline control</w:t></w:r></w:sdtContent></w:sdt>` +
		`<w:smartTag><w:r><w:t xml:space="preserve"> tag</w:t></w:r></w:smartTag></w:p>` +
		`<mc:AlternateContent><mc:Choice Requires="w14">` + para("choice") + `</mc:Choice><mc:Fallback>` + para("fallback") + `</mc:Fallback></mc:AlternateContent>` +
		`<w:customXml>` + para("custom") + `</w:customXml><w:sectPr/>`
	s, _ := readStream(t, testDoc{body: body}, Limits{})
	want := []string{"in control", "inline control tag", "choice", "custom"}
	if got := texts(s.blocks); !reflect.DeepEqual(got, want) {
		t.Fatalf("texts = %q, want %q", got, want)
	}
}

func TestCountsAndEmptyParagraphs(t *testing.T) {
	body := para("Two words") + `<w:p/>` + para("three", " more words") + para("   ")
	s, result := readStream(t, testDoc{body: body}, Limits{})
	if len(s.blocks) != 4 {
		t.Fatalf("blocks = %q; an empty paragraph is kept", texts(s.blocks))
	}
	if s.end.Words != 5 || s.end.Paragraphs != 2 || s.end.Characters != 9+16+3 || s.end.Truncated {
		t.Errorf("end = %+v", s.end)
	}
	if result.Words != 5 || result.Blocks != 4 || result.Paragraphs != 2 {
		t.Errorf("result = %+v", result)
	}
}

func TestHeadersFootersAndCommentsAreNamed(t *testing.T) {
	rels := `<Relationship Id="rH" Type="` + relNS + `/header" Target="header1.xml"/>` +
		`<Relationship Id="rF" Type="` + relNS + `/footer" Target="footer1.xml"/>` +
		`<Relationship Id="rC" Type="` + relNS + `/comments" Target="comments.xml"/>`
	s, _ := readStream(t, testDoc{body: para("x"), rels: rels}, Limits{})
	want := []string{warnHeadersFooters, warnComments}
	if !reflect.DeepEqual(s.end.Warnings, want) {
		t.Errorf("warnings = %q, want %q", s.end.Warnings, want)
	}
}

func TestDocmMacrosAreNeverOpened(t *testing.T) {
	d := testDoc{body: para("macro document"), files: map[string]string{"word/vbaProject.bin": "\xd0\xcf\x11\xe0 not really OLE"},
		rels: `<Relationship Id="rVba" Type="http://schemas.microsoft.com/office/2006/relationships/vbaProject" Target="vbaProject.bin"/>`}
	data := d.zip(t)
	var out strings.Builder
	_, err := Convert(ConvertParams{Source: strings.NewReader(string(data)), Size: int64(len(data)), Format: FormatDOCM, Out: &out})
	if err != nil {
		t.Fatal(err)
	}
	s := parseStream(t, out.String())
	if s.meta.Format != FormatDOCM || !reflect.DeepEqual(texts(s.blocks), []string{"macro document"}) {
		t.Errorf("stream = %+v", s)
	}
}

func TestPackageWithoutRelationships(t *testing.T) {
	// A package missing _rels/.rels still has its body where Word puts it.
	d := testDoc{body: para("found")}
	data := d.zip(t)
	stripped := rezip(t, data, func(name string) bool { return name != "_rels/.rels" })
	out, _, err := convert(t, stripped, OutputNDJSON, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if got := texts(parseStream(t, out).blocks); !reflect.DeepEqual(got, []string{"found"}) {
		t.Errorf("texts = %q", got)
	}
}

func TestFormatForName(t *testing.T) {
	for name, want := range map[string]Format{"a.docx": FormatDOCX, "B.DOCM": FormatDOCM, "c.doc": "", "d": "", "e.docx.txt": ""} {
		got, ok := FormatForName(name)
		if got != want || ok != (want != "") {
			t.Errorf("FormatForName(%q) = %q, %v", name, got, ok)
		}
	}
}
