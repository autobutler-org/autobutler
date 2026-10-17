package v0_files_test

import (
	"archive/zip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	v0_files "github.com/autobutler-org/quark/internal/server/api/v0/files"
	"github.com/autobutler-org/quark/pkg/util/accessutil"
)

// writeDocx packages a one-paragraph DOCX holding a table at rel.
func writeDocx(t *testing.T, filesDir, rel string) {
	t.Helper()
	const relNS = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	full := filepath.Join(filesDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(full)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for name, content := range map[string]string{
		"_rels/.rels": `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="` + relNS + `/officeDocument" Target="word/document.xml"/></Relationships>`,
		"word/document.xml": `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
			`<w:p><w:r><w:t>Minutes of the meeting</w:t></w:r></w:p>` +
			`<w:tbl><w:tr><w:tc><w:p><w:r><w:t>a</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>b</w:t></w:r></w:p></w:tc></w:tr></w:tbl>` +
			`</w:body></w:document>`,
	} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestConvertDocument_WritesAQdocTheCallerOwns(t *testing.T) {
	h := newAccessHarness(t, false)
	writeDocx(t, h.filesDir, "shared/minutes.docx")
	h.grant(t, "shared", accessutil.Write)

	w := h.post("/api/v0/files/convert/document?filePath=shared/minutes.docx")
	if w.Code != http.StatusOK {
		t.Fatalf("convert = %d: %s", w.Code, w.Body.String())
	}
	var got v0_files.ConvertDocumentJSON
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Path != "shared/minutes.qdoc" || got.Truncated || got.Words != 6 {
		t.Errorf("response = %+v", got)
	}
	if len(got.Warnings) != 1 || !strings.Contains(got.Warnings[0], "table") {
		t.Errorf("warnings = %q, want the flattened table named", got.Warnings)
	}
	raw, err := os.ReadFile(filepath.Join(h.filesDir, "shared", "minutes.qdoc"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "Minutes of the meeting") || !strings.Contains(string(raw), `{"insert":"\t"}`) {
		t.Errorf(".qdoc = %s", raw)
	}
	if lv := h.levels(t)["shared/minutes.qdoc"]; lv != "owner" {
		t.Errorf("level of the new .qdoc = %q, want owner", lv)
	}

	expectCodes(t, http.StatusConflict, map[string]*httptest.ResponseRecorder{
		"again": h.post("/api/v0/files/convert/document?filePath=shared/minutes.docx"),
	})
	expectCodes(t, http.StatusOK, map[string]*httptest.ResponseRecorder{
		"overwrite": h.post("/api/v0/files/convert/document?filePath=shared/minutes.docx&overwrite=true"),
	})
}

func TestConvertDocument_Refusals(t *testing.T) {
	h := newAccessHarness(t, false)
	writeDocx(t, h.filesDir, "shared/minutes.docx")
	writeDocx(t, h.filesDir, "other/secret.docx")
	writeFixture(t, h.filesDir, "shared/fake.docx")
	h.grant(t, "shared", accessutil.Read)

	expectCodes(t, http.StatusForbidden, map[string]*httptest.ResponseRecorder{
		"read-only share": h.post("/api/v0/files/convert/document?filePath=shared/minutes.docx"),
	})
	expectCodes(t, http.StatusNotFound, map[string]*httptest.ResponseRecorder{
		"no access": h.post("/api/v0/files/convert/document?filePath=other/secret.docx"),
	})
	h.grant(t, "shared", accessutil.Write)
	expectCodes(t, http.StatusBadRequest, map[string]*httptest.ResponseRecorder{
		"not what it claims": h.post("/api/v0/files/convert/document?filePath=shared/fake.docx"),
		"no filePath":        h.post("/api/v0/files/convert/document"),
	})
	for _, rel := range []string{"shared/minutes.qdoc", "shared/fake.qdoc", "other/secret.qdoc"} {
		if _, err := os.Stat(filepath.Join(h.filesDir, filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Errorf("a refused conversion wrote %s", rel)
		}
	}
}
