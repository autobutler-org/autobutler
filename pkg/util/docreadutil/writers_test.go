package docreadutil

// cspell:ignore abstractNum abstractNumId gridSpan ilvl jc lvl lvlText numFmt numId numPr pPr pStyle rPr styleId

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestBlockJSONShapes(t *testing.T) {
	cases := []struct {
		block Block
		want  string
	}{
		{Block{Type: BlockHeading, Level: 2}, `{"type":"heading","level":2,"runs":[]}`},
		{Block{Type: BlockParagraph, Align: "justify", Runs: []Run{{Text: "Revenue "}, {Text: "rose", Bold: true}, {Text: " again.", Link: "https://example.com"}}},
			`{"type":"paragraph","align":"justify","runs":[{"text":"Revenue "},{"text":"rose","bold":true},{"text":" again.","link":"https://example.com"}]}`},
		{Block{Type: BlockListItem, Marker: "•", Runs: []Run{{Text: "x"}}}, `{"type":"listItem","ordered":false,"level":0,"marker":"•","runs":[{"text":"x"}]}`},
		{Block{Type: BlockTable, Rows: []TableRow{{Cells: []TableCell{{ColSpan: 2, RowSpan: 1}}}, {}}},
			`{"type":"table","rows":[{"cells":[{"colSpan":2,"rowSpan":1,"blocks":[]}]},{"cells":[]}]}`},
		{Block{Type: BlockImage, Part: "word/media/image1.png", Alt: "Chart of revenue", Width: 640, Height: 360},
			`{"type":"image","part":"word/media/image1.png","alt":"Chart of revenue","width":640,"height":360}`},
		{Block{Type: BlockImage, External: true, Alt: "Logo", Part: ""}, `{"type":"image","external":true,"alt":"Logo"}`},
		{Block{Type: BlockOmitted, What: "embedded object", Runs: []Run{{Text: "ignored"}}}, `{"type":"omitted","what":"embedded object"}`},
		{Block{Type: BlockPageBreak, Level: 3}, `{"type":"pageBreak"}`},
	}
	for _, c := range cases {
		got, err := json.Marshal(c.block)
		if err != nil || string(got) != c.want {
			t.Errorf("Marshal(%+v) =\n%s, %v\nwant\n%s", c.block, got, err, c.want)
		}
	}
}

func TestNDJSONOfAnEmptyDocument(t *testing.T) {
	out, _, err := convert(t, testDoc{}.zip(t), OutputNDJSON, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"type":"meta","version":1,"format":"docx"}` + "\n" +
		`{"type":"end","truncated":false,"words":0,"characters":0,"paragraphs":0,"warnings":[]}` + "\n"
	if out != want {
		t.Errorf("stream =\n%s\nwant\n%s", out, want)
	}
}

// deltaDoc is a .qdoc's JSON.
type deltaDoc struct {
	Ops []map[string]any `json:"ops"`
}

// readDelta converts a document to a Delta and parses it.
func readDelta(t *testing.T, d testDoc) (deltaDoc, ConvertResult) {
	t.Helper()
	out, result, err := convert(t, d.zip(t), OutputDelta, Limits{})
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	var doc deltaDoc
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("the Delta is not JSON: %v\n%s", err, out)
	}
	return doc, result
}

// op is a Delta op as it parses back.
func op(insert string, attrs map[string]any) map[string]any {
	o := map[string]any{"insert": insert}
	if attrs != nil {
		o["attributes"] = attrs
	}
	return o
}

func TestDeltaOfAnEmptyDocument(t *testing.T) {
	out, result, err := convert(t, testDoc{}.zip(t), OutputDelta, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if out != `{"ops":[{"insert":"\n"}]}` || len(result.Warnings) != 0 {
		t.Errorf("Delta = %s, warnings %q", out, result.Warnings)
	}
}

func TestDelta(t *testing.T) {
	styles := `<w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/></w:style>`
	numbering := `<w:abstractNum w:abstractNumId="0"><w:lvl w:ilvl="0"><w:start w:val="1"/><w:numFmt w:val="decimal"/><w:lvlText w:val="%1."/></w:lvl>` +
		`<w:lvl w:ilvl="1"><w:numFmt w:val="bullet"/><w:lvlText w:val="-"/></w:lvl></w:abstractNum><w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num>`
	rels := `<Relationship Id="rLink" Type="` + relNS + `/hyperlink" Target="https://example.com" TargetMode="External"/>` +
		`<Relationship Id="rImg" Type="` + relNS + `/image" Target="media/a.png"/>`
	body := `<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Big</w:t><w:br/><w:t>Title</w:t></w:r></w:p>` +
		`<w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:b/><w:i/></w:rPr><w:t>bold</w:t></w:r>` +
		`<w:hyperlink r:id="rLink"><w:r><w:t xml:space="preserve"> link</w:t></w:r></w:hyperlink>` +
		`<w:r><w:rPr><w:vertAlign w:val="superscript"/></w:rPr><w:t>2</w:t></w:r></w:p>` +
		listed("1", "0", "first") + listed("1", "1", "sub") +
		`<w:tbl><w:tr>` + cell("", "a") + `<w:tc><w:p><w:r><w:t>b</w:t><w:br/><w:t>c</w:t></w:r></w:p>` + para("d") + `</w:tc></w:tr>` +
		`<w:tr>` + cell(`<w:gridSpan w:val="2"/>`, "wide") + `</w:tr></w:tbl>` +
		`<w:p>` + drawing(`r:embed="rImg"`, "") + `<w:r><w:br w:type="page"/><w:object/></w:r></w:p>`
	doc, result := readDelta(t, testDoc{styles: styles, numbering: numbering, rels: rels, body: body,
		files: map[string]string{"word/media/a.png": "png"}})
	h1 := map[string]any{"header": 1.0}
	want := []map[string]any{
		op("Big", nil), op("\n", h1), op("Title", nil), op("\n", h1),
		op("bold", map[string]any{"bold": true, "italic": true}), op(" link", map[string]any{"link": "https://example.com"}),
		op("2", map[string]any{"script": "super"}), op("\n", map[string]any{"align": "center"}),
		op("first", nil), op("\n", map[string]any{"list": "ordered"}),
		op("sub", nil), op("\n", map[string]any{"list": "bullet", "indent": 1.0}),
		op("a", nil), op("\t", nil), op("b c", nil), op(" ", nil), op("d", nil), op("\n", nil),
		op("wide", nil), op("\n", nil),
	}
	if !reflect.DeepEqual(doc.Ops, want) {
		got, _ := json.Marshal(doc.Ops)
		wantJSON, _ := json.Marshal(want)
		t.Errorf("ops =\n%s\nwant\n%s", got, wantJSON)
	}
	wantWarnings := []string{"1 embedded object not shown", "1 table flattened to text", "1 image left out"}
	if !reflect.DeepEqual(result.Warnings, wantWarnings) {
		t.Errorf("warnings = %q, want %q", result.Warnings, wantWarnings)
	}
}

func TestDeltaFlattensImagesAndTablesInsideCells(t *testing.T) {
	rels := `<Relationship Id="rImg" Type="` + relNS + `/image" Target="media/a.png"/>`
	body := `<w:tbl><w:tr><w:tc><w:p>` + drawing(`r:embed="rImg"`, "") + `</w:p>` + nested(1, "inner") + `</w:tc></w:tr></w:tbl>`
	doc, result := readDelta(t, testDoc{rels: rels, body: body, files: map[string]string{"word/media/a.png": "png"}})
	want := []map[string]any{op("inner", nil), op("\n", nil)}
	if !reflect.DeepEqual(doc.Ops, want) {
		t.Errorf("ops = %v, want %v", doc.Ops, want)
	}
	if want := []string{"2 tables flattened to text", "1 image left out"}; !reflect.DeepEqual(result.Warnings, want) {
		t.Errorf("warnings = %q, want %q", result.Warnings, want)
	}
}
