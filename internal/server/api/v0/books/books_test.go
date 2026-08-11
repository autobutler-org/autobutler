package v0_books_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	v0_books "github.com/autobutler-org/autobutler/internal/server/api/v0/books"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// newBooksEngine creates a gin engine with the books routes registered.
// It points HOME at a temp dir so GetCirrusDir() resolves to a controlled location.
func newBooksEngine(t *testing.T) (*gin.Engine, string) {
	t.Helper()

	// Redirect the data directory to a temp location.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Pre-create the cirrus directory that GetCirrusDir builds.
	cirrusDir := filepath.Join(tmpHome, "autobutler", "data", "cirrus")
	if err := os.MkdirAll(cirrusDir, 0755); err != nil {
		t.Fatalf("failed to create cirrus dir: %v", err)
	}

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	group := engine.Group("/api/v0")
	serverutil.RegisterRouterWithGroup(group, v0_books.NewRouter())
	return engine, cirrusDir
}

func doGet(engine *gin.Engine, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

// TestListBooks_EmptyDir verifies that an empty cirrus directory returns 200
// with an empty JSON array.
func TestListBooks_EmptyDir(t *testing.T) {
	engine, _ := newBooksEngine(t)

	w := doGet(engine, "/api/v0/books")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result []any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse body as JSON array: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty array, got %d items", len(result))
	}
}

// TestListBooks_WithEpub verifies that an .epub file in the cirrus directory
// appears in the response with the correct fields populated.
func TestListBooks_WithEpub(t *testing.T) {
	engine, cirrusDir := newBooksEngine(t)

	// Create a minimal .epub placeholder.
	epubPath := filepath.Join(cirrusDir, "test-book.epub")
	if err := os.WriteFile(epubPath, []byte("fake epub content"), 0644); err != nil {
		t.Fatalf("failed to write epub: %v", err)
	}

	w := doGet(engine, "/api/v0/books")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 book, got %d", len(result))
	}

	book := result[0]
	if book["fileName"] != "test-book.epub" {
		t.Errorf("unexpected fileName: %v", book["fileName"])
	}
	if book["type"] == nil || book["type"] == "" {
		t.Errorf("expected type field to be set")
	}
	if book["mtime"] == nil {
		t.Errorf("expected mtime field to be set")
	}
	if book["size"] == nil {
		t.Errorf("expected size field to be set")
	}
}

// TestListBooks_IgnoresNonBookFiles verifies that non-book files (e.g. .txt, .png)
// are excluded from the response and only recognised book types are returned.
func TestListBooks_IgnoresNonBookFiles(t *testing.T) {
	engine, cirrusDir := newBooksEngine(t)

	// Write one book and two non-book files.
	files := map[string][]byte{
		"novel.epub": []byte("epub"),
		"readme.txt": []byte("not a book"),
		"cover.png":  []byte("not a book"),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(cirrusDir, name), content, 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	w := doGet(engine, "/api/v0/books")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 book (epub only), got %d: %v", len(result), result)
	}
	if result[0]["fileName"] != "novel.epub" {
		t.Errorf("unexpected fileName: %v", result[0]["fileName"])
	}
}

// TestListBooks_PdfIncluded verifies that .pdf files are also returned (PDF is a
// supported book type alongside epub).
func TestListBooks_PdfIncluded(t *testing.T) {
	engine, cirrusDir := newBooksEngine(t)

	if err := os.WriteFile(filepath.Join(cirrusDir, "manual.pdf"), []byte("%PDF"), 0644); err != nil {
		t.Fatalf("failed to write pdf: %v", err)
	}

	w := doGet(engine, "/api/v0/books")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 book (pdf), got %d", len(result))
	}
	if result[0]["fileName"] != "manual.pdf" {
		t.Errorf("unexpected fileName: %v", result[0]["fileName"])
	}
}

// TestListBooks_SubdirectoryRecursion verifies that books nested in subdirectories
// are also returned.
func TestListBooks_SubdirectoryRecursion(t *testing.T) {
	engine, cirrusDir := newBooksEngine(t)

	subDir := filepath.Join(cirrusDir, "fiction", "scifi")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "dune.epub"), []byte("dune"), 0644); err != nil {
		t.Fatalf("failed to write epub: %v", err)
	}

	w := doGet(engine, "/api/v0/books")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 book, got %d", len(result))
	}
	relPath, _ := result[0]["relPath"].(string)
	if relPath == "" {
		t.Errorf("expected relPath to be set")
	}
}
