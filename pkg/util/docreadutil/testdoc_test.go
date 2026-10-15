package docreadutil

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

// The namespaces a test document declares on its root.
const testNamespaces = `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" ` +
	`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" ` +
	`xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" ` +
	`xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" ` +
	`xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture" ` +
	`xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006" ` +
	`xmlns:v="urn:schemas-microsoft-com:vml" xmlns:o="urn:schemas-microsoft-com:office:office" ` +
	`xmlns:m="http://schemas.openxmlformats.org/officeDocument/2006/math"`

const relNS = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"

// testDoc is a DOCX package built for a test: each field is a part's
// content inside its root element, and an empty field leaves the part out.
type testDoc struct {
	body      string
	styles    string
	numbering string
	footnotes string
	endnotes  string
	title     string
	// rels are extra <Relationship> elements of the document part.
	rels string
	// files are extra entries, by name.
	files map[string]string
}

// zip builds the package.
func (d testDoc) zip(t testing.TB) []byte {
	t.Helper()
	pkgRels := `<Relationship Id="rId1" Type="` + relNS + `/officeDocument" Target="word/document.xml"/>`
	files := []struct{ name, body string }{
		{"[Content_Types].xml", `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"/>`},
		{"word/document.xml", `<?xml version="1.0" encoding="UTF-8"?><w:document ` + testNamespaces + `><w:body>` + d.body + `</w:body></w:document>`},
	}
	docRels := d.rels
	addPart := func(id, relType, target, rootName, content string) {
		if content == "" {
			return
		}
		docRels += `<Relationship Id="` + id + `" Type="` + relNS + `/` + relType + `" Target="` + target + `"/>`
		files = append(files, struct{ name, body string }{"word/" + target, `<w:` + rootName + ` ` + testNamespaces + `>` + content + `</w:` + rootName + `>`})
	}
	addPart("rIdStyles", "styles", "styles.xml", "styles", d.styles)
	addPart("rIdNumbering", "numbering", "numbering.xml", "numbering", d.numbering)
	addPart("rIdFootnotes", "footnotes", "footnotes.xml", "footnotes", d.footnotes)
	addPart("rIdEndnotes", "endnotes", "endnotes.xml", "endnotes", d.endnotes)
	if d.title != "" {
		pkgRels += `<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>`
		files = append(files, struct{ name, body string }{"docProps/core.xml",
			`<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>` + d.title + `</dc:title></cp:coreProperties>`})
	}
	files = append(files,
		struct{ name, body string }{"_rels/.rels", `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` + pkgRels + `</Relationships>`},
		struct{ name, body string }{"word/_rels/document.xml.rels", `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` + docRels + `</Relationships>`},
	)
	for name, body := range d.files {
		files = append(files, struct{ name, body string }{name, body})
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

// convert runs Convert over a package held in memory.
func convert(t testing.TB, data []byte, output Output, limits Limits) (string, ConvertResult, error) {
	t.Helper()
	var out strings.Builder
	result, err := Convert(ConvertParams{
		Source: bytes.NewReader(data), Size: int64(len(data)), Format: FormatDOCX,
		Output: output, Out: &out, Limits: limits,
	})
	return out.String(), result, err
}

// stream is a parsed NDJSON stream.
type stream struct {
	meta   metaLine
	blocks []Block
	end    endLine
}

// readStream converts a document to NDJSON and parses it, failing the test
// unless it has a meta line first, an end line last, and blocks between.
func readStream(t testing.TB, d testDoc, limits Limits) (stream, ConvertResult) {
	t.Helper()
	out, result, err := convert(t, d.zip(t), OutputNDJSON, limits)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	return parseStream(t, out), result
}

// parseStream parses an NDJSON stream.
func parseStream(t testing.TB, out string) stream {
	t.Helper()
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("stream has %d lines, want meta and end at least:\n%s", len(lines), out)
	}
	var s stream
	if err := json.Unmarshal([]byte(lines[0]), &s.meta); err != nil || s.meta.Type != "meta" {
		t.Fatalf("first line %q is not meta: %v", lines[0], err)
	}
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &s.end); err != nil || s.end.Type != "end" {
		t.Fatalf("last line %q is not end: %v", lines[len(lines)-1], err)
	}
	for _, line := range lines[1 : len(lines)-1] {
		var b Block
		if err := json.Unmarshal([]byte(line), &b); err != nil {
			t.Fatalf("line %q: %v", line, err)
		}
		s.blocks = append(s.blocks, b)
	}
	return s
}

// plain is a text block's text, its runs joined.
func plain(b Block) string {
	var sb strings.Builder
	for _, r := range b.Runs {
		sb.WriteString(r.Text)
	}
	return sb.String()
}

// texts is every block's plain text, with a table's cells joined by "|"
// and other blocks named by type.
func texts(blocks []Block) []string {
	var out []string
	for _, b := range blocks {
		switch b.Type {
		case BlockTable:
			var cells []string
			for _, row := range b.Rows {
				for _, c := range row.Cells {
					cells = append(cells, strings.Join(texts(c.Blocks), "/"))
				}
			}
			out = append(out, "table["+strings.Join(cells, "|")+"]")
		case BlockImage, BlockOmitted, BlockPageBreak:
			out = append(out, string(b.Type)+":"+b.What+b.Part)
		default:
			out = append(out, plain(b))
		}
	}
	return out
}

// para is a paragraph of plain runs.
func para(text ...string) string {
	var sb strings.Builder
	sb.WriteString("<w:p>")
	for _, t := range text {
		sb.WriteString(`<w:r><w:t xml:space="preserve">` + t + `</w:t></w:r>`)
	}
	sb.WriteString("</w:p>")
	return sb.String()
}

// rezip copies a zip, keeping the entries keep says to.
func rezip(t testing.TB, data []byte, keep func(name string) bool) []byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range zr.File {
		if !keep(f.Name) {
			continue
		}
		if err := zw.Copy(f); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
