package v1_webdav

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newTestRouter creates a gin engine with the WebDAV handler mounted at /dav/.
func newTestRouter(t *testing.T) (*gin.Engine, string) {
	t.Helper()
	root := t.TempDir()

	router := gin.New()
	handler := NewHandler(root)
	for _, method := range WebDAVMethods() {
		router.Handle(method, "/dav/*filepath", handler)
	}
	return router, root
}

func TestNewHandler(t *testing.T) {
	root := t.TempDir()
	handler := NewHandler(root)
	if handler == nil {
		t.Fatal("NewHandler returned nil")
	}
}

func TestPROPFIND_Root(t *testing.T) {
	router, _ := newTestRouter(t)

	req := httptest.NewRequest("PROPFIND", "/dav/", nil)
	req.Header.Set("Depth", "0")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusMultiStatus {
		t.Fatalf("PROPFIND root: expected 207, got %d; body: %s", w.Code, w.Body.String())
	}
}

func TestPUT_then_GET(t *testing.T) {
	router, root := newTestRouter(t)

	// PUT a file
	content := "hello webdav"
	req := httptest.NewRequest(http.MethodPut, "/dav/test.txt", strings.NewReader(content))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated && w.Code != http.StatusNoContent {
		t.Fatalf("PUT: expected 201 or 204, got %d; body: %s", w.Code, w.Body.String())
	}

	// Verify on disk
	data, err := os.ReadFile(filepath.Join(root, "test.txt"))
	if err != nil {
		t.Fatalf("file not written to disk: %v", err)
	}
	if string(data) != content {
		t.Fatalf("file content mismatch: got %q, want %q", string(data), content)
	}

	// GET it back
	req = httptest.NewRequest(http.MethodGet, "/dav/test.txt", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET: expected 200, got %d", w.Code)
	}
	body, _ := io.ReadAll(w.Result().Body)
	if string(body) != content {
		t.Fatalf("GET body mismatch: got %q, want %q", string(body), content)
	}
}

func TestMKCOL(t *testing.T) {
	router, root := newTestRouter(t)

	req := httptest.NewRequest("MKCOL", "/dav/newdir/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("MKCOL: expected 201, got %d; body: %s", w.Code, w.Body.String())
	}

	info, err := os.Stat(filepath.Join(root, "newdir"))
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected a directory")
	}
}

func TestDELETE(t *testing.T) {
	router, root := newTestRouter(t)

	// Create a file first
	path := filepath.Join(root, "todelete.txt")
	if err := os.WriteFile(path, []byte("bye"), 0644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/dav/todelete.txt", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE: expected 204, got %d; body: %s", w.Code, w.Body.String())
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("file still exists after DELETE")
	}
}

func TestIsWebDAVPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/dav/", true},
		{"/dav/some/file.txt", true},
		{"/dav", true},
		{"/api/v1/auth/login", false},
		{"/", false},
		{"/davinci", false},
	}
	for _, tt := range tests {
		if got := IsWebDAVPath(tt.path); got != tt.want {
			t.Errorf("IsWebDAVPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
