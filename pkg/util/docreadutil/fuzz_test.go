package docreadutil

// cspell:ignore abstractNum abstractNumId fldChar fldCharType footnoteReference gridSpan ilvl instrText lvl lvlText numFmt numId numPr pPr rPr sdt sdtContent tbl tc tcPr tr vMerge

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

// fuzzLimits keep a fuzzed document cheap to read.
var fuzzLimits = Limits{MaxBlocks: 2000, MaxOutputBytes: 1 << 20, MaxHeldBytes: 1 << 20, MaxDefinitions: 500}

// readWithoutRecover converts a package the way Convert does, without
// Convert's recover, so a panic fails the fuzz target rather than becoming
// an error, and checks that what it writes is well formed.
func readWithoutRecover(t *testing.T, data []byte, output Output) {
	readFormatWithoutRecover(t, data, output, FormatDOCX, readDocx)
}

// readFormatWithoutRecover is readWithoutRecover for a format's reader.
func readFormatWithoutRecover(t *testing.T, data []byte, output Output, format Format, read func(*emitter, io.ReaderAt, int64) error) {
	var out bytes.Buffer
	e := newEmitter(ConvertParams{Format: format, Output: output, Out: &out, Limits: fuzzLimits})
	err := read(e, bytes.NewReader(data), int64(len(data)))
	if err != nil && !errors.Is(err, errTruncated) {
		if !errors.Is(err, ErrMalformed) && !errors.Is(err, ErrTooLarge) {
			t.Fatalf("an error that is neither malformed nor too large: %v", err)
		}
		return
	}
	if err := e.finish(); err != nil {
		t.Fatal(err)
	}
	if output == OutputDelta {
		var doc deltaDoc
		if err := json.Unmarshal(out.Bytes(), &doc); err != nil || len(doc.Ops) == 0 {
			t.Fatalf("the Delta is not a document: %v\n%s", err, out.String())
		}
		return
	}
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	for i, line := range lines {
		var b map[string]any
		if err := json.Unmarshal([]byte(line), &b); err != nil {
			t.Fatalf("line %d is not JSON: %v\n%s", i, err, line)
		}
	}
	if !strings.HasPrefix(lines[0], `{"type":"meta"`) || !strings.HasPrefix(lines[len(lines)-1], `{"type":"end"`) {
		t.Fatalf("the stream does not run from meta to end:\n%s", out.String())
	}
}

// fuzzBodies are document bodies that reach every part of the reader.
var fuzzBodies = []string{
	para("Hello", " world"),
	`<w:p><w:pPr><w:pStyle w:val="Heading1"/><w:numPr><w:ilvl w:val="1"/><w:numId w:val="1"/></w:numPr><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:b/><w:vertAlign w:val="superscript"/></w:rPr><w:t>x</w:t><w:br w:type="page"/><w:tab/></w:r></w:p>`,
	`<w:tbl><w:tr><w:tc><w:tcPr><w:gridSpan w:val="2"/><w:vMerge w:val="restart"/></w:tcPr>` + para("a") + `</w:tc></w:tr><w:tr><w:tc><w:tcPr><w:vMerge/></w:tcPr>` + nested(2, "b") + `</w:tc></w:tr></w:tbl>`,
	`<w:p><w:r><w:fldChar w:fldCharType="begin"/></w:r><w:r><w:instrText>X</w:instrText></w:r><w:r><w:fldChar w:fldCharType="separate"/></w:r><w:r><w:t>r</w:t><w:footnoteReference w:id="1"/></w:r></w:p>`,
	`<w:sdt><w:sdtContent><w:p>` + drawing(`r:embed="rImg"`, "alt") + `<w:hyperlink r:id="rLink"><w:r><w:t>l</w:t></w:r></w:hyperlink></w:p></w:sdtContent></w:sdt><w:altChunk/>`,
}

const fuzzStyles = `<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:basedOn w:val="Normal"/><w:rPr><w:i/></w:rPr></w:style>`

const fuzzNumbering = `<w:abstractNum w:abstractNumId="0"><w:lvl w:ilvl="1"><w:start w:val="3"/><w:numFmt w:val="lowerRoman"/><w:lvlText w:val="%1.%2"/></w:lvl></w:abstractNum><w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num>`

const fuzzRels = `<Relationship Id="rImg" Type="` + relNS + `/image" Target="media/a.png"/>` +
	`<Relationship Id="rLink" Type="` + relNS + `/hyperlink" Target="https://example.com" TargetMode="External"/>`

// FuzzConvert reads arbitrary bytes as a DOCX package.
func FuzzConvert(f *testing.F) {
	for _, body := range fuzzBodies {
		f.Add(testDoc{body: body, styles: fuzzStyles, numbering: fuzzNumbering, rels: fuzzRels,
			footnotes: `<w:footnote w:id="1">` + para("note") + `</w:footnote>`,
			files:     map[string]string{"word/media/a.png": "png"}}.zip(f))
	}
	f.Add([]byte("PK\x05\x06"))
	f.Fuzz(func(t *testing.T, data []byte) {
		readWithoutRecover(t, data, OutputNDJSON)
		readWithoutRecover(t, data, OutputDelta)
	})
}

// FuzzDocumentXML reads arbitrary document, style and numbering XML in a
// well-formed package, where the reader's own logic meets hostile input
// rather than the zip layer.
func FuzzDocumentXML(f *testing.F) {
	for _, body := range fuzzBodies {
		f.Add(body, fuzzStyles, fuzzNumbering)
	}
	f.Fuzz(func(t *testing.T, body, styles, numbering string) {
		data := testDoc{body: body, styles: styles, numbering: numbering, rels: fuzzRels,
			files: map[string]string{"word/media/a.png": "png"}}.zip(t)
		readWithoutRecover(t, data, OutputNDJSON)
		readWithoutRecover(t, data, OutputDelta)
	})
}
