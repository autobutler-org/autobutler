package fileutil_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/eventbus"
	"github.com/autobutler-org/quark/pkg/util/fileutil"
	"github.com/autobutler-org/quark/pkg/vfs"
)

// pngBytes stands in for an image part; nothing here decodes it.
const pngBytes = "\x89PNG\r\n\x1a\nnot really"

// documentParts is a one-paragraph DOCX with one image in its media folder
// and a non-image part beside it.
func documentParts(body string) map[string]string {
	const rels = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	return map[string]string{
		"[Content_Types].xml": `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"/>`,
		"_rels/.rels": `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="` + rels + `/officeDocument" Target="word/document.xml"/></Relationships>`,
		"word/document.xml": `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
			body + `</w:body></w:document>`,
		"word/_rels/document.xml.rels": `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"/>`,
		"word/media/image1.png":        pngBytes,
		"word/media/notes.txt":         "not an image",
	}
}

const helloBody = `<w:p><w:r><w:t>Hello there world</w:t></w:r></w:p>`

// newDocumentFixture is a files namespace holding report.docx.
func newDocumentFixture(t *testing.T, parts map[string]string) (string, vfs.Registry) {
	t.Helper()
	root := t.TempDir()
	seedParts(t, root, "report.docx", parts)
	fsys, err := vfs.NewLocalVFS(root, "files")
	if err != nil {
		t.Fatal(err)
	}
	registry := vfs.NewRegistry()
	if err := registry.Register(vfs.Namespace{ID: "files", MountPath: "/"}, fsys); err != nil {
		t.Fatal(err)
	}
	return root, registry
}

func convertDocumentParams(registry vfs.Registry) fileutil.ConvertDocumentParams {
	return fileutil.ConvertDocumentParams{
		Ctx:      context.Background(),
		Registry: registry,
		EventBus: eventbus.New(),
		FilePath: "report.docx",
	}
}

func TestConvertDocumentToQdocWritesASiblingDoc(t *testing.T) {
	root, registry := newDocumentFixture(t, documentParts(helloBody))

	result, err := fileutil.ConvertDocumentToQdoc(convertDocumentParams(registry))
	if err != nil {
		t.Fatalf("ConvertDocumentToQdoc: %v", err)
	}
	if result.Path != "report.qdoc" || result.Replaced || result.Words != 3 {
		t.Errorf("result = %+v, want report.qdoc, new, 3 words", result)
	}
	raw, err := os.ReadFile(filepath.Join(root, "report.qdoc"))
	if err != nil {
		t.Fatal(err)
	}
	var delta struct {
		Ops []struct {
			Insert string `json:"insert"`
		} `json:"ops"`
	}
	if err := json.Unmarshal(raw, &delta); err != nil {
		t.Fatalf("not a Delta: %v\n%s", err, raw)
	}
	var text strings.Builder
	for _, op := range delta.Ops {
		text.WriteString(op.Insert)
	}
	if text.String() != "Hello there world\n" {
		t.Errorf("text = %q", text.String())
	}
	if got := leftovers(t, root); len(got) != 0 {
		t.Errorf("left behind %v", got)
	}
}

func TestConvertDocumentToQdocRefusesToClobberUnlessAsked(t *testing.T) {
	root, registry := newDocumentFixture(t, documentParts(helloBody))
	existing := filepath.Join(root, "report.qdoc")
	if err := os.WriteFile(existing, []byte(`{"ops":[{"insert":"mine\n"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := fileutil.ConvertDocumentToQdoc(convertDocumentParams(registry))
	if !errors.Is(err, vfs.ErrConflict) {
		t.Fatalf("err = %v, want a conflict", err)
	}
	if raw, _ := os.ReadFile(existing); !strings.Contains(string(raw), "mine") {
		t.Fatal("a refused conversion changed the existing .qdoc")
	}

	params := convertDocumentParams(registry)
	params.Overwrite = true
	result, err := fileutil.ConvertDocumentToQdoc(params)
	if err != nil || !result.Replaced {
		t.Fatalf("overwrite = %+v, %v; want replaced", result, err)
	}
	if raw, _ := os.ReadFile(existing); !strings.Contains(string(raw), "Hello") {
		t.Errorf("overwrite left %s", raw)
	}
}

func TestConvertDocumentToQdocRejectsOtherExtensions(t *testing.T) {
	_, registry := newDocumentFixture(t, documentParts(helloBody))
	params := convertDocumentParams(registry)
	params.FilePath = "notes.txt"
	var unsupported *fileutil.UnsupportedError
	if _, err := fileutil.ConvertDocumentToQdoc(params); !errors.As(err, &unsupported) {
		t.Errorf("err = %v, want UnsupportedError", err)
	}
}

func TestConvertDocumentToQdocLeavesNothingBehindWhenTheDocumentIsBroken(t *testing.T) {
	root, registry := newDocumentFixture(t, documentParts(`<w:p><w:r><w:t>unclosed`))

	_, err := fileutil.ConvertDocumentToQdoc(convertDocumentParams(registry))
	var unsupported *fileutil.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("err = %v, want UnsupportedError", err)
	}
	if _, err := os.Stat(filepath.Join(root, "report.qdoc")); !os.IsNotExist(err) {
		t.Error("a failed conversion wrote report.qdoc")
	}
	if got := leftovers(t, root); len(got) != 0 {
		t.Errorf("left behind %v", got)
	}
}

func TestPreviewDocumentStreamsBlocks(t *testing.T) {
	_, registry := newDocumentFixture(t, documentParts(helloBody))
	var out bytes.Buffer

	result, err := fileutil.PreviewDocument(fileutil.PreviewDocumentParams{
		Ctx: context.Background(), Registry: registry, FilePath: "report.docx", Out: &out,
	})
	if err != nil {
		t.Fatalf("PreviewDocument: %v", err)
	}
	if result.Truncated {
		t.Error("a one-line document came back truncated")
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 || !strings.Contains(lines[0], `"type":"meta"`) ||
		!strings.Contains(lines[1], "Hello there world") || !strings.Contains(lines[2], `"type":"end"`) {
		t.Errorf("stream = %q", lines)
	}
}

func TestPreviewDocumentWritesNothingForAFileThatIsNotADocument(t *testing.T) {
	root, registry := newDocumentFixture(t, documentParts(helloBody))
	if err := os.WriteFile(filepath.Join(root, "fake.docx"), []byte("plain text"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer

	_, err := fileutil.PreviewDocument(fileutil.PreviewDocumentParams{
		Ctx: context.Background(), Registry: registry, FilePath: "fake.docx", Out: &out,
	})
	var unsupported *fileutil.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Errorf("err = %v, want UnsupportedError", err)
	}
	if out.Len() != 0 {
		t.Errorf("wrote %q before failing", out.String())
	}
}

func TestOpenDocumentPart(t *testing.T) {
	_, registry := newDocumentFixture(t, documentParts(helloBody))
	open := func(part string) (fileutil.OpenDocumentPartResult, error) {
		return fileutil.OpenDocumentPart(fileutil.OpenDocumentPartParams{
			Ctx: context.Background(), Registry: registry, FilePath: "report.docx", Part: part,
		})
	}

	got, err := open("word/media/image1.png")
	if err != nil {
		t.Fatalf("open the image: %v", err)
	}
	body, err := io.ReadAll(got.Reader)
	if closeErr := got.Close(); err != nil || closeErr != nil {
		t.Fatalf("read %v, close %v", err, closeErr)
	}
	if string(body) != pngBytes || got.ContentType != "image/png" {
		t.Errorf("served %q as %s", body, got.ContentType)
	}

	for _, part := range []string{
		"word/document.xml",               // not in the media folder
		"word/media/notes.txt",            // not an image
		"word/media/missing.png",          // not in the package
		"word/media/../document.xml",      // climbs out
		"word/media/../../etc/passwd.png", // climbs out of the package
		"/word/media/image1.png",          // absolute
		"word\\media\\image1.png",         // backslashes
		"word/media/",                     // the folder itself
		"",
	} {
		var notFound *fileutil.NotFoundError
		if _, err := open(part); !errors.As(err, &notFound) {
			t.Errorf("part %q: err = %v, want NotFoundError", part, err)
		}
	}
}
