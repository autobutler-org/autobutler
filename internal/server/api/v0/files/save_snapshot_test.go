package v0_files_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"slices"
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
