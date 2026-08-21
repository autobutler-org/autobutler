package v1_vfs_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	v1_vfs "github.com/autobutler-org/quark/internal/server/api/v1/vfs"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/vfs"
	"github.com/gin-gonic/gin"
)

func newVFSEngine(t *testing.T, reg vfs.Registry) *gin.Engine {
	t.Helper()
	deps := deputil.NewDependencies().WithVFSRegistry(reg)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c = ctxutil.With(c, "deps", deps)
		c.Next()
	})
	group := engine.Group("/api/v1")
	serverutil.RegisterRouterWithGroup(group, v1_vfs.NewRouter())
	return engine
}

func newVFSEngineWithNS(t *testing.T, nsID string) (*gin.Engine, *vfs.MemVFS) {
	t.Helper()
	reg := vfs.NewRegistry()
	mem := vfs.NewMemVFS(nsID)
	if err := reg.Register(vfs.Namespace{ID: nsID, Description: "test"}, mem); err != nil {
		t.Fatalf("Register: %v", err)
	}
	return newVFSEngine(t, reg), mem
}

func doVFSReq(engine *gin.Engine, method, path string, body io.Reader, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

// --- List Namespaces ---------------------------------------------------------

// TestListNamespaces_Empty verifies GET /vfs returns [] when no namespaces registered.
func TestListNamespaces_Empty(t *testing.T) {
	engine := newVFSEngine(t, vfs.NewRegistry())

	w := doVFSReq(engine, http.MethodGet, "/api/v1/vfs", nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /vfs returned %d: %s", w.Code, w.Body.String())
	}
	var ns []v1_vfs.NamespaceJSON
	json.Unmarshal(w.Body.Bytes(), &ns)
	if ns == nil {
		t.Error("expected empty slice, not null")
	}
	if len(ns) != 0 {
		t.Errorf("expected 0 namespaces, got %d", len(ns))
	}
}

// TestListNamespaces_ReturnsRegistered verifies a registered namespace appears
// in GET /vfs.
func TestListNamespaces_ReturnsRegistered(t *testing.T) {
	reg := vfs.NewRegistry()
	reg.Register(vfs.Namespace{ID: "docs", Description: "Documents"}, vfs.NewMemVFS("docs"))
	engine := newVFSEngine(t, reg)

	w := doVFSReq(engine, http.MethodGet, "/api/v1/vfs", nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /vfs returned %d: %s", w.Code, w.Body.String())
	}
	var ns []v1_vfs.NamespaceJSON
	json.Unmarshal(w.Body.Bytes(), &ns)
	if len(ns) != 1 {
		t.Fatalf("expected 1 namespace, got %d; body: %s", len(ns), w.Body.String())
	}
	if ns[0].ID != "docs" {
		t.Errorf("ID = %q; want 'docs'", ns[0].ID)
	}
	if ns[0].Description != "Documents" {
		t.Errorf("Description = %q; want 'Documents'", ns[0].Description)
	}
}

// --- VFS Read ----------------------------------------------------------------

// TestVFSRead_UnknownNamespaceReturns404 verifies GET /vfs/:ns/*path returns
// 404 for a namespace that doesn't exist.
func TestVFSRead_UnknownNamespaceReturns404(t *testing.T) {
	engine := newVFSEngine(t, vfs.NewRegistry())

	w := doVFSReq(engine, http.MethodGet, "/api/v1/vfs/ghost/file.txt", nil, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// TestVFSRead_ListsRootDirectory verifies GET /vfs/:ns/ lists the root
// directory of the namespace (empty initially).
func TestVFSRead_ListsRootDirectory(t *testing.T) {
	engine, _ := newVFSEngineWithNS(t, "data")

	w := doVFSReq(engine, http.MethodGet, "/api/v1/vfs/data/", nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /vfs/data/ returned %d: %s", w.Code, w.Body.String())
	}
	var entries []v1_vfs.FileInfoJSON
	json.Unmarshal(w.Body.Bytes(), &entries)
	if entries == nil {
		t.Error("expected empty slice, not null")
	}
}

// TestVFSWrite_ThenRead verifies PUT /vfs/:ns/*path stores a file and GET
// streams it back with the correct content.
func TestVFSWrite_ThenRead(t *testing.T) {
	engine, _ := newVFSEngineWithNS(t, "files")

	// Write.
	content := []byte("hello from the VFS test")
	w := doVFSReq(engine, http.MethodPut, "/api/v1/vfs/files/hello.txt",
		bytes.NewReader(content),
		map[string]string{"Content-Type": "text/plain"},
	)
	if w.Code != http.StatusCreated {
		t.Fatalf("PUT returned %d: %s", w.Code, w.Body.String())
	}

	// Read back.
	w2 := doVFSReq(engine, http.MethodGet, "/api/v1/vfs/files/hello.txt", nil, nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("GET returned %d: %s", w2.Code, w2.Body.String())
	}
	got := w2.Body.String()
	if got != string(content) {
		t.Errorf("read body = %q; want %q", got, content)
	}
}

// TestVFSRead_Stat verifies GET /vfs/:ns/*path?stat=true returns FileInfo JSON
// instead of streaming the file.
func TestVFSRead_Stat(t *testing.T) {
	engine, mem := newVFSEngineWithNS(t, "imgs")

	// Seed a file directly in the MemVFS.
	content := []byte("img data")
	if err := mem.Write(nil, "/photo.jpg", bytes.NewReader(content),
		vfs.WriteOptions{ContentType: "image/jpeg"}); err != nil {
		t.Fatalf("MemVFS.Write: %v", err)
	}

	w := doVFSReq(engine, http.MethodGet, "/api/v1/vfs/imgs/photo.jpg?stat=true", nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET?stat=true returned %d: %s", w.Code, w.Body.String())
	}
	var fi v1_vfs.FileInfoJSON
	json.Unmarshal(w.Body.Bytes(), &fi)
	if fi.Name != "photo.jpg" {
		t.Errorf("Name = %q; want 'photo.jpg'", fi.Name)
	}
	if fi.Size != int64(len(content)) {
		t.Errorf("Size = %d; want %d", fi.Size, len(content))
	}
}

// TestVFSWrite_ConflictWithIfNoneMatch verifies that a second PUT with
// If-None-Match: * returns 409 when the file already exists.
func TestVFSWrite_ConflictWithIfNoneMatch(t *testing.T) {
	engine, _ := newVFSEngineWithNS(t, "store")

	write := func() *httptest.ResponseRecorder {
		return doVFSReq(engine, http.MethodPut, "/api/v1/vfs/store/unique.txt",
			strings.NewReader("data"),
			map[string]string{
				"Content-Type":  "text/plain",
				"If-None-Match": "*",
			},
		)
	}

	w1 := write()
	if w1.Code != http.StatusCreated {
		t.Fatalf("first PUT returned %d", w1.Code)
	}
	w2 := write()
	if w2.Code != http.StatusConflict {
		t.Errorf("second PUT with If-None-Match: * returned %d; want 409", w2.Code)
	}
}

// TestVFSDelete_RemovesFile verifies DELETE /vfs/:ns/*path removes a file so
// it's no longer readable.
func TestVFSDelete_RemovesFile(t *testing.T) {
	engine, mem := newVFSEngineWithNS(t, "trash")

	if err := mem.Write(nil, "/gone.txt", strings.NewReader("bye"),
		vfs.WriteOptions{ContentType: "text/plain"}); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	w := doVFSReq(engine, http.MethodDelete, "/api/v1/vfs/trash/gone.txt", nil, nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE returned %d: %s", w.Code, w.Body.String())
	}

	w2 := doVFSReq(engine, http.MethodGet, "/api/v1/vfs/trash/gone.txt", nil, nil)
	if w2.Code != http.StatusNotFound {
		t.Errorf("GET after DELETE returned %d; want 404", w2.Code)
	}
}

// TestVFSMkdir_CreatesDirectory verifies POST /vfs/:ns/*path creates a
// directory listable by GET.
func TestVFSMkdir_CreatesDirectory(t *testing.T) {
	engine, _ := newVFSEngineWithNS(t, "dirs")

	w := doVFSReq(engine, http.MethodPost, "/api/v1/vfs/dirs/subdir/", nil, nil)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST mkdir returned %d: %s", w.Code, w.Body.String())
	}

	// Write a file inside it to confirm it exists.
	doVFSReq(engine, http.MethodPut, "/api/v1/vfs/dirs/subdir/file.txt",
		strings.NewReader("content"),
		map[string]string{"Content-Type": "text/plain"},
	)

	// List the directory.
	w2 := doVFSReq(engine, http.MethodGet, "/api/v1/vfs/dirs/subdir/", nil, nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("GET subdir/ returned %d: %s", w2.Code, w2.Body.String())
	}
	var entries []v1_vfs.FileInfoJSON
	json.Unmarshal(w2.Body.Bytes(), &entries)
	if len(entries) == 0 {
		t.Error("expected at least 1 entry in subdir")
	}
}

// TestVFSRead_UnknownFileReturns404 verifies GET /vfs/:ns/*path returns 404
// for a file that doesn't exist in a known namespace.
func TestVFSRead_UnknownFileReturns404(t *testing.T) {
	engine, _ := newVFSEngineWithNS(t, "ns1")

	w := doVFSReq(engine, http.MethodGet, "/api/v1/vfs/ns1/doesnotexist.txt", nil, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
