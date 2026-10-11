package v0_files_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/storageutil"
)

// snapshotsOf reads every snapshot in the version store of dir/name.
func snapshotsOf(t *testing.T, filesDir, dir, name string) []string {
	t.Helper()
	snaps, err := filepath.Glob(filepath.Join(filesDir, dir, storageutil.VersionsDirName, name, "*.snap"))
	if err != nil {
		t.Fatal(err)
	}
	contents := make([]string, 0, len(snaps))
	for _, snap := range snaps {
		b, err := os.ReadFile(snap)
		if err != nil {
			t.Fatal(err)
		}
		contents = append(contents, string(b))
	}
	return contents
}

// A chunked upload that replaces an editor document snapshots it first, the
// way the multipart upload does (#2992). Any other file is replaced without
// one.
func TestChunkedOverwriteSnapshotsEditorDocuments(t *testing.T) {
	t.Parallel()
	e, filesDir := newStorageVFSTestEngine(t)
	content := []byte("second, sent in chunks")
	for _, name := range []string{"deck.qslide", "notes.txt"} {
		if w := uploadFile(t, e, "/api/v0/files/upload/docs", name, "first"); w.Code != http.StatusOK {
			t.Fatalf("upload %s returned %d: %s", name, w.Code, w.Body.String())
		}
		w := openSession(t, e, map[string]any{
			"rootDir": "docs", "fileName": name, "totalSize": len(content), "overwrite": true,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("open session for %s returned %d: %s", name, w.Code, w.Body.String())
		}
		if got := decodeSession(t, uploadInChunks(t, e, decodeSession(t, w).SessionID, content, 8)); !got.Complete {
			t.Fatalf("chunked overwrite of %s reported %+v, want complete", name, got)
		}
		assertContent(t, filepath.Join(filesDir, "docs", name), string(content))
	}
	if got := snapshotsOf(t, filesDir, "docs", "deck.qslide"); !slices.Equal(got, []string{"first"}) {
		t.Errorf("deck.qslide snapshots = %q, want the content the upload replaced", got)
	}
	if got := snapshotsOf(t, filesDir, "docs", "notes.txt"); len(got) != 0 {
		t.Errorf("notes.txt snapshots = %q; only editor documents are versioned", got)
	}
}

// Versions live on the internal device only, so replacing an editor document
// on a device addressed by serial snapshots nothing, by either upload.
func TestOverwriteOnADeviceTakesNoSnapshot(t *testing.T) {
	t.Parallel()
	h := newDeviceUploadHarness(t)
	base := "/api/v0/files/upload/docs?serial=" + testUsbSerial
	if w := uploadFile(t, h.engine, base, "deck.qslide", "first"); w.Code != http.StatusOK {
		t.Fatalf("upload returned %d: %s", w.Code, w.Body.String())
	}
	if w := uploadFile(t, h.engine, base+"&overwrite=true", "deck.qslide", "second"); w.Code != http.StatusOK {
		t.Fatalf("overwrite returned %d: %s", w.Code, w.Body.String())
	}
	content := []byte("third")
	w := openSession(t, h.engine, map[string]any{
		"rootDir": "docs", "fileName": "deck.qslide", "totalSize": len(content), "overwrite": true, "serial": testUsbSerial,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("open session returned %d: %s", w.Code, w.Body.String())
	}
	uploadInChunks(t, h.engine, decodeSession(t, w).SessionID, content, len(content))
	assertContent(t, filepath.Join(h.usbDir, "docs", "deck.qslide"), "third")
	for _, filesDir := range []string{h.usbDir, h.internalDir} {
		assertAbsent(t, filepath.Join(filesDir, "docs", storageutil.VersionsDirName))
	}
}

// A move that lands on an editor document snapshots what it replaces (#2992).
// A move to a free name, or over any other file, snapshots nothing.
func TestMoveOverAnEditorDocumentSnapshotsIt(t *testing.T) {
	t.Parallel()
	e, filesDir := newStorageVFSTestEngine(t)
	for name, content := range map[string]string{
		"deck.qslide": "replaced deck", "incoming.qslide": "moved deck",
		"notes.txt": "replaced notes", "incoming.txt": "moved notes", "free.qslide": "renamed",
	} {
		if w := uploadFile(t, e, "/api/v0/files/upload/docs", name, content); w.Code != http.StatusOK {
			t.Fatalf("upload %s returned %d: %s", name, w.Code, w.Body.String())
		}
	}
	for _, move := range [][2]string{
		{"incoming.qslide", "deck.qslide"}, {"incoming.txt", "notes.txt"}, {"free.qslide", "fresh.qslide"},
	} {
		body, _ := json.Marshal(map[string]string{"oldFilePath": "docs/" + move[0], "newFilePath": "docs/" + move[1]})
		if w := doRequest(e, http.MethodPut, "/api/v0/files", bytes.NewReader(body), "application/json"); w.Code != http.StatusOK {
			t.Fatalf("move %v returned %d: %s", move, w.Code, w.Body.String())
		}
	}
	assertContent(t, filepath.Join(filesDir, "docs", "deck.qslide"), "moved deck")
	if got := snapshotsOf(t, filesDir, "docs", "deck.qslide"); !slices.Equal(got, []string{"replaced deck"}) {
		t.Errorf("deck.qslide snapshots = %q, want the content the move replaced", got)
	}
	for _, name := range []string{"notes.txt", "fresh.qslide"} {
		if got := snapshotsOf(t, filesDir, "docs", name); len(got) != 0 {
			t.Errorf("%s snapshots = %q, want none", name, got)
		}
	}
}

// minimalWorkbook is the smallest .xlsx the converter accepts: one sheet with
// one cell.
func minimalWorkbook(t *testing.T) string {
	t.Helper()
	const rels = `<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`
	const relType = `http://schemas.openxmlformats.org/officeDocument/2006/relationships/`
	parts := map[string]string{
		"_rels/.rels": rels + `<Relationship Id="rId1" Type="` + relType + `officeDocument" Target="xl/workbook.xml"/></Relationships>`,
		"xl/workbook.xml": `<?xml version="1.0"?><workbook xmlns:r="` + strings.TrimSuffix(relType, "/") + `">` +
			`<sheets><sheet name="Ledger" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": rels + `<Relationship Id="rId1" Type="` + relType + `worksheet" Target="worksheets/sheet1.xml"/></Relationships>`,
		"xl/worksheets/sheet1.xml": `<?xml version="1.0"?><worksheet><sheetData>` +
			`<row r="1"><c r="A1" t="inlineStr"><is><t>Item</t></is></c></row></sheetData></worksheet>`,
	}
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
	return buf.String()
}

// Converting a workbook over an existing .qsheet snapshots the sheet it
// replaces, as every other overwrite does (#2992). A conversion to a free name
// snapshots nothing.
func TestConvertOverAnExistingSheetSnapshotsIt(t *testing.T) {
	t.Parallel()
	e, filesDir := newStorageVFSTestEngine(t)
	workbook := minimalWorkbook(t)
	for _, name := range []string{"book.xlsx", "fresh.xlsx"} {
		if w := uploadFile(t, e, "/api/v0/files/upload/docs", name, workbook); w.Code != http.StatusOK {
			t.Fatalf("upload %s returned %d: %s", name, w.Code, w.Body.String())
		}
	}
	if w := uploadFile(t, e, "/api/v0/files/upload/docs", "book.qsheet", "hand edited sheet"); w.Code != http.StatusOK {
		t.Fatalf("upload sheet returned %d: %s", w.Code, w.Body.String())
	}
	for _, name := range []string{"book", "fresh"} {
		w := doRequest(e, http.MethodPost, "/api/v0/files/convert/xlsx?overwrite=true&filePath=docs/"+name+".xlsx", nil, "")
		if w.Code != http.StatusOK {
			t.Fatalf("convert %s returned %d: %s", name, w.Code, w.Body.String())
		}
	}
	if got := snapshotsOf(t, filesDir, "docs", "book.qsheet"); !slices.Equal(got, []string{"hand edited sheet"}) {
		t.Errorf("book.qsheet snapshots = %q, want the sheet the conversion replaced", got)
	}
	if got := snapshotsOf(t, filesDir, "docs", "fresh.qsheet"); len(got) != 0 {
		t.Errorf("fresh.qsheet snapshots = %q, want none", got)
	}
}

// Versions live on the internal device only, so a conversion that replaces a
// sheet on a device addressed by serial snapshots nothing.
func TestConvertOnADeviceTakesNoSnapshot(t *testing.T) {
	t.Parallel()
	h := newDeviceUploadHarness(t)
	upload := "/api/v0/files/upload/docs?serial=" + testUsbSerial
	for name, content := range map[string]string{"book.xlsx": minimalWorkbook(t), "book.qsheet": "hand edited sheet"} {
		if w := uploadFile(t, h.engine, upload, name, content); w.Code != http.StatusOK {
			t.Fatalf("upload %s returned %d: %s", name, w.Code, w.Body.String())
		}
	}
	w := doRequest(h.engine, http.MethodPost,
		"/api/v0/files/convert/xlsx?overwrite=true&filePath=docs/book.xlsx&serial="+testUsbSerial, nil, "")
	if w.Code != http.StatusOK {
		t.Fatalf("convert returned %d: %s", w.Code, w.Body.String())
	}
	for _, filesDir := range []string{h.usbDir, h.internalDir} {
		assertAbsent(t, filepath.Join(filesDir, "docs", storageutil.VersionsDirName))
	}
}
