package v1_files_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

// statPath calls GET /api/v1/cirrus/stat and returns the decoded JSON body.
func statPath(t *testing.T, engine *gin.Engine, filePath string) (map[string]any, int) {
	t.Helper()
	url := fmt.Sprintf("/api/v1/cirrus/stat?filePath=%s", filePath)
	w := doRequest(engine, http.MethodGet, url, nil, "")
	if w.Code == http.StatusOK {
		var result map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("failed to decode stat response: %v", err)
		}
		return result, w.Code
	}
	return nil, w.Code
}

// --- stat endpoint: regular files ---

func TestStatFile_AbdocFile(t *testing.T) {
	engine, cirrusDir := newTestEngine(t)
	if err := os.WriteFile(filepath.Join(cirrusDir, "note.abdoc"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	body, code := statPath(t, engine, "note.abdoc")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if isDir, _ := body["isDir"].(bool); isDir {
		t.Error("expected isDir=false for an abdoc file")
	}
	if ft, _ := body["fileType"].(string); ft != "abdoc" {
		t.Errorf("expected fileType %q, got %q", "abdoc", ft)
	}
}

func TestStatFile_AbsheetFile(t *testing.T) {
	engine, cirrusDir := newTestEngine(t)
	if err := os.WriteFile(filepath.Join(cirrusDir, "budget.absheet"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	body, code := statPath(t, engine, "budget.absheet")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if isDir, _ := body["isDir"].(bool); isDir {
		t.Error("expected isDir=false for an absheet file")
	}
	if ft, _ := body["fileType"].(string); ft != "absheet" {
		t.Errorf("expected fileType %q, got %q", "absheet", ft)
	}
}

func TestStatFile_ImageFile(t *testing.T) {
	engine, cirrusDir := newTestEngine(t)
	if err := os.WriteFile(filepath.Join(cirrusDir, "photo.jpg"), []byte("\xFF\xD8"), 0644); err != nil {
		t.Fatal(err)
	}

	body, code := statPath(t, engine, "photo.jpg")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if isDir, _ := body["isDir"].(bool); isDir {
		t.Error("expected isDir=false for an image file")
	}
	if ft, _ := body["fileType"].(string); ft != "image" {
		t.Errorf("expected fileType %q, got %q", "image", ft)
	}
}

func TestStatFile_VideoFile(t *testing.T) {
	engine, cirrusDir := newTestEngine(t)
	if err := os.WriteFile(filepath.Join(cirrusDir, "clip.mp4"), []byte("ftyp"), 0644); err != nil {
		t.Fatal(err)
	}

	body, code := statPath(t, engine, "clip.mp4")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if isDir, _ := body["isDir"].(bool); isDir {
		t.Error("expected isDir=false for a video file")
	}
	if ft, _ := body["fileType"].(string); ft != "video" {
		t.Errorf("expected fileType %q, got %q", "video", ft)
	}
}

// --- stat endpoint: plain directories ---

func TestStatFile_PlainDirectory(t *testing.T) {
	engine, cirrusDir := newTestEngine(t)
	if err := os.Mkdir(filepath.Join(cirrusDir, "photos"), 0755); err != nil {
		t.Fatal(err)
	}

	body, code := statPath(t, engine, "photos")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if isDir, _ := body["isDir"].(bool); !isDir {
		t.Error("expected isDir=true for a plain directory")
	}
	if ft, _ := body["fileType"].(string); ft != "folder" {
		t.Errorf("expected fileType %q, got %q", "folder", ft)
	}
}

// --- stat endpoint: folders with misleading extensions ---
// These are the core regression cases. A folder named like a known file
// extension must be identified as a directory, not as that file type.

func TestStatFile_FolderNamedLikeAbdoc(t *testing.T) {
	engine, cirrusDir := newTestEngine(t)
	if err := os.Mkdir(filepath.Join(cirrusDir, "things.abdoc"), 0755); err != nil {
		t.Fatal(err)
	}

	body, code := statPath(t, engine, "things.abdoc")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if isDir, _ := body["isDir"].(bool); !isDir {
		t.Error("expected isDir=true: a folder named things.abdoc must not be treated as a document")
	}
	if ft, _ := body["fileType"].(string); ft != "folder" {
		t.Errorf("expected fileType %q, got %q", "folder", ft)
	}
}

func TestStatFile_FolderNamedLikeAbsheet(t *testing.T) {
	engine, cirrusDir := newTestEngine(t)
	if err := os.Mkdir(filepath.Join(cirrusDir, "data.absheet"), 0755); err != nil {
		t.Fatal(err)
	}

	body, code := statPath(t, engine, "data.absheet")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if isDir, _ := body["isDir"].(bool); !isDir {
		t.Error("expected isDir=true: a folder named data.absheet must not be treated as a spreadsheet")
	}
	if ft, _ := body["fileType"].(string); ft != "folder" {
		t.Errorf("expected fileType %q, got %q", "folder", ft)
	}
}

func TestStatFile_FolderNamedLikeImage(t *testing.T) {
	engine, cirrusDir := newTestEngine(t)
	if err := os.Mkdir(filepath.Join(cirrusDir, "photo.jpg"), 0755); err != nil {
		t.Fatal(err)
	}

	body, code := statPath(t, engine, "photo.jpg")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if isDir, _ := body["isDir"].(bool); !isDir {
		t.Error("expected isDir=true: a folder named photo.jpg must not be treated as an image")
	}
	if ft, _ := body["fileType"].(string); ft != "folder" {
		t.Errorf("expected fileType %q, got %q", "folder", ft)
	}
}

func TestStatFile_FolderNamedLikeVideo(t *testing.T) {
	engine, cirrusDir := newTestEngine(t)
	if err := os.Mkdir(filepath.Join(cirrusDir, "clip.mp4"), 0755); err != nil {
		t.Fatal(err)
	}

	body, code := statPath(t, engine, "clip.mp4")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if isDir, _ := body["isDir"].(bool); !isDir {
		t.Error("expected isDir=true: a folder named clip.mp4 must not be treated as a video")
	}
	if ft, _ := body["fileType"].(string); ft != "folder" {
		t.Errorf("expected fileType %q, got %q", "folder", ft)
	}
}

// --- stat endpoint: nested paths ---

func TestStatFile_NestedFile(t *testing.T) {
	engine, cirrusDir := newTestEngine(t)
	sub := filepath.Join(cirrusDir, "docs")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "report.abdoc"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	body, code := statPath(t, engine, "docs/report.abdoc")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if isDir, _ := body["isDir"].(bool); isDir {
		t.Error("expected isDir=false for a nested abdoc file")
	}
	if ft, _ := body["fileType"].(string); ft != "abdoc" {
		t.Errorf("expected fileType %q, got %q", "abdoc", ft)
	}
}

func TestStatFile_NestedFolderNamedLikeFile(t *testing.T) {
	engine, cirrusDir := newTestEngine(t)
	sub := filepath.Join(cirrusDir, "projects")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(sub, "archive.zip"), 0755); err != nil {
		t.Fatal(err)
	}

	body, code := statPath(t, engine, "projects/archive.zip")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if isDir, _ := body["isDir"].(bool); !isDir {
		t.Error("expected isDir=true: nested folder named archive.zip must not be treated as an archive file")
	}
	if ft, _ := body["fileType"].(string); ft != "folder" {
		t.Errorf("expected fileType %q, got %q", "folder", ft)
	}
}

// --- stat endpoint: not found ---

func TestStatFile_NotFound(t *testing.T) {
	engine, _ := newTestEngine(t)

	_, code := statPath(t, engine, "nonexistent.abdoc")
	if code != http.StatusNotFound {
		t.Errorf("expected 404 for missing path, got %d", code)
	}
}
