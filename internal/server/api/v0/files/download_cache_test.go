package v0_files_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestDownload_CacheControlNoCache verifies that downloads forbid a browser
// from serving the body without revalidating.
//
// The response carries Last-Modified. Without Cache-Control a browser is free
// to apply heuristic freshness (RFC 9111 §4.2.2) and answer from its own cache
// without asking, which is how a .qsheet saved from the editor reopened
// showing its pre-save contents.
func TestDownload_CacheControlNoCache(t *testing.T) {
	engine, vfsRoot := newVFSTestEngine(t)

	if err := os.WriteFile(filepath.Join(vfsRoot, "sheet.qsheet"), []byte("v1"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v0/files/download?filePath=sheet.qsheet", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control: got %q, want %q", cc, "no-cache")
	}
	if lm := w.Header().Get("Last-Modified"); lm == "" {
		t.Error("Last-Modified missing: without it no-cache costs a full body on every read")
	}
}

// TestDownload_RevalidatesTo304 verifies the other half of the no-cache trade:
// an unchanged file still answers a conditional request with 304 and no body,
// so forbidding blind reuse does not mean re-sending every file every time.
func TestDownload_RevalidatesTo304(t *testing.T) {
	engine, vfsRoot := newVFSTestEngine(t)

	if err := os.WriteFile(filepath.Join(vfsRoot, "sheet.qsheet"), []byte("v1"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	first := httptest.NewRecorder()
	engine.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/v0/files/download?filePath=sheet.qsheet", nil))
	lastModified := first.Header().Get("Last-Modified")
	if lastModified == "" {
		t.Fatal("Last-Modified missing from the first response")
	}

	conditional := httptest.NewRequest(http.MethodGet, "/api/v0/files/download?filePath=sheet.qsheet", nil)
	conditional.Header.Set("If-Modified-Since", lastModified)
	second := httptest.NewRecorder()
	engine.ServeHTTP(second, conditional)

	if second.Code != http.StatusNotModified {
		t.Errorf("unchanged file: got %d, want 304", second.Code)
	}
	if body := second.Body.Len(); body != 0 {
		t.Errorf("304 carried a %d-byte body", body)
	}
}
