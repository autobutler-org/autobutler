package v1_vfs_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	v1_vfs "github.com/autobutler-org/autobutler/internal/server/api/v1/vfs"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/vfs"
	"github.com/gin-gonic/gin"
)

// ─── helpers ────────────────────────────────────────────────────────────────

const metaSchema = `
CREATE TABLE IF NOT EXISTS vfs_metadata (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    namespace   TEXT NOT NULL,
    path        TEXT NOT NULL,
    key         TEXT NOT NULL,
    value       TEXT NOT NULL,
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE (namespace, path, key)
);
CREATE INDEX IF NOT EXISTS idx_vfs_metadata_ns_path ON vfs_metadata (namespace, path);
CREATE INDEX IF NOT EXISTS idx_vfs_metadata_ns_key  ON vfs_metadata (namespace, key);
`

func newTestMetaDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if _, err := db.Exec(metaSchema); err != nil {
		t.Fatalf("schema: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func newVFSEngineWithMeta(t *testing.T, nsID string) (*gin.Engine, *vfs.MemVFS) {
	t.Helper()
	reg := vfs.NewRegistry()
	mem := vfs.NewMemVFS(nsID)
	if err := reg.Register(vfs.Namespace{ID: nsID, Description: "test"}, mem); err != nil {
		t.Fatalf("Register: %v", err)
	}
	db := newTestMetaDB(t)
	store := vfs.NewSQLiteMetadataStore(db)
	deps := deputil.NewDependencies().
		WithVFSRegistry(reg).
		WithMetadataStore(store)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c = ctxutil.With(c, "deps", deps)
		c.Next()
	})
	group := engine.Group("/api/v1")
	serverutil.RegisterRouterWithGroup(group, v1_vfs.NewRouter())
	return engine, mem
}

func doReq(engine *gin.Engine, method, path string, body io.Reader, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

// ─── read: file streaming ───────────────────────────────────────────────────

// TestVFSRead_StreamsFileBytes verifies that a stored file's raw bytes are
// returned with the correct Content-Type header.
func TestVFSRead_StreamsFileBytes(t *testing.T) {
	engine, mem := newVFSEngineWithMeta(t, "docs")
	content := "hello from vfs"
	if err := mem.Write(context.Background(), "/hello.txt",
		strings.NewReader(content),
		vfs.WriteOptions{ContentType: "text/plain", ExpectedSize: int64(len(content))}); err != nil {
		t.Fatalf("Put: %v", err)
	}

	w := doReq(engine, http.MethodGet, "/api/v1/vfs/docs/hello.txt", nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got != content {
		t.Errorf("expected body %q, got %q", content, got)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("expected text/plain Content-Type, got %q", ct)
	}
}

// TestVFSRead_RecursiveListing verifies that ?recursive=true returns nested entries.
func TestVFSRead_RecursiveListing(t *testing.T) {
	engine, mem := newVFSEngineWithMeta(t, "docs")
	if err := mem.Write(context.Background(), "/sub/deep.txt",
		strings.NewReader("nested"),
		vfs.WriteOptions{ContentType: "text/plain", ExpectedSize: 6}); err != nil {
		t.Fatalf("Put: %v", err)
	}

	w := doReq(engine, http.MethodGet, "/api/v1/vfs/docs/?recursive=true", nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var entries []v1_vfs.FileInfoJSON
	if err := json.Unmarshal(w.Body.Bytes(), &entries); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	found := false
	for _, e := range entries {
		if strings.Contains(e.Path, "deep.txt") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected deep.txt in recursive listing, got: %v", entries)
	}
}

// TestVFSRead_MimePrefixFilter verifies that ?mime_prefix= filters entries by MIME type.
func TestVFSRead_MimePrefixFilter(t *testing.T) {
	engine, mem := newVFSEngineWithMeta(t, "docs")
	for _, f := range []struct {
		path, mime, body string
	}{
		{"/doc.txt", "text/plain", "text"},
		{"/img.png", "image/png", "png"},
	} {
		if err := mem.Write(context.Background(), f.path,
			strings.NewReader(f.body),
			vfs.WriteOptions{ContentType: f.mime, ExpectedSize: int64(len(f.body))}); err != nil {
			t.Fatalf("Put %s: %v", f.path, err)
		}
	}

	w := doReq(engine, http.MethodGet, "/api/v1/vfs/docs/?mime_prefix=image/", nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var entries []v1_vfs.FileInfoJSON
	json.Unmarshal(w.Body.Bytes(), &entries)
	if len(entries) != 1 {
		t.Fatalf("expected 1 image entry, got %d: %v", len(entries), entries)
	}
	if !strings.Contains(entries[0].Path, "img.png") {
		t.Errorf("expected img.png in filtered listing, got %q", entries[0].Path)
	}
}

// TestVFSRead_ImplicitDirList verifies that a path without trailing slash that
// resolves to a directory returns a listing (not an error).
func TestVFSRead_ImplicitDirList(t *testing.T) {
	engine, mem := newVFSEngineWithMeta(t, "docs")
	if err := mem.Write(context.Background(), "/sub/a.txt",
		strings.NewReader("a"),
		vfs.WriteOptions{ContentType: "text/plain", ExpectedSize: 1}); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Request /sub (no trailing slash) — handler should detect it's a dir and list it.
	w := doReq(engine, http.MethodGet, "/api/v1/vfs/docs/sub", nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var entries []v1_vfs.FileInfoJSON
	if err := json.Unmarshal(w.Body.Bytes(), &entries); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected at least one entry in implicit dir listing")
	}
}

// ─── meta: GET / PUT / DELETE ────────────────────────────────────────────────

// TestVFSMeta_GetEmpty verifies that /_meta on a new path returns an empty object.
func TestVFSMeta_GetEmpty(t *testing.T) {
	engine, _ := newVFSEngineWithMeta(t, "docs")

	w := doReq(engine, http.MethodGet, "/api/v1/vfs/docs/nonexistent.txt/_meta", nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var meta map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &meta); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if len(meta) != 0 {
		t.Errorf("expected empty metadata, got %v", meta)
	}
}

// TestVFSMeta_SetThenGet verifies that metadata written with PUT is readable via GET.
func TestVFSMeta_SetThenGet(t *testing.T) {
	engine, _ := newVFSEngineWithMeta(t, "docs")

	payload, _ := json.Marshal(map[string]any{"favorite": true, "rating": 5})
	w := doReq(engine, http.MethodPut, "/api/v1/vfs/docs/photo.jpg/_meta",
		bytes.NewReader(payload), map[string]string{"Content-Type": "application/json"})
	if w.Code != http.StatusOK {
		t.Fatalf("PUT /_meta returned %d: %s", w.Code, w.Body.String())
	}

	w = doReq(engine, http.MethodGet, "/api/v1/vfs/docs/photo.jpg/_meta", nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /_meta returned %d: %s", w.Code, w.Body.String())
	}
	var meta map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &meta); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if _, ok := meta["favorite"]; !ok {
		t.Errorf("expected 'favorite' key in metadata, got %v", meta)
	}
	if _, ok := meta["rating"]; !ok {
		t.Errorf("expected 'rating' key in metadata, got %v", meta)
	}
}

// TestVFSMeta_DeleteKeys verifies that specific keys can be removed via DELETE /_meta.
func TestVFSMeta_DeleteKeys(t *testing.T) {
	engine, _ := newVFSEngineWithMeta(t, "docs")

	// Write two keys.
	payload, _ := json.Marshal(map[string]any{"keep": "yes", "remove": "bye"})
	doReq(engine, http.MethodPut, "/api/v1/vfs/docs/file.txt/_meta",
		bytes.NewReader(payload), map[string]string{"Content-Type": "application/json"})

	// Delete only "remove".
	del, _ := json.Marshal([]string{"remove"})
	w := doReq(engine, http.MethodDelete, "/api/v1/vfs/docs/file.txt/_meta",
		bytes.NewReader(del), map[string]string{"Content-Type": "application/json"})
	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE /_meta returned %d: %s", w.Code, w.Body.String())
	}

	// Verify "keep" remains, "remove" is gone.
	w = doReq(engine, http.MethodGet, "/api/v1/vfs/docs/file.txt/_meta", nil, nil)
	var meta map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &meta)
	if _, ok := meta["keep"]; !ok {
		t.Error("expected 'keep' key to remain after DELETE /_meta")
	}
	if _, ok := meta["remove"]; ok {
		t.Error("expected 'remove' key to be deleted")
	}
}

// TestVFSMeta_BadJSON verifies that PUT /_meta with invalid JSON returns 400.
func TestVFSMeta_BadJSON(t *testing.T) {
	engine, _ := newVFSEngineWithMeta(t, "docs")

	w := doReq(engine, http.MethodPut, "/api/v1/vfs/docs/file.txt/_meta",
		strings.NewReader("{not valid json}"),
		map[string]string{"Content-Type": "application/json"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad JSON, got %d: %s", w.Code, w.Body.String())
	}
}
