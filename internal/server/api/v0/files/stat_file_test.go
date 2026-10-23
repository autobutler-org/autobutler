package v0_files_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

// statPath calls GET /api/v0/files/stat and returns the decoded JSON body.
func statPath(t *testing.T, engine *gin.Engine, filePath string) (map[string]any, int) {
	t.Helper()
	url := fmt.Sprintf("/api/v0/files/stat?filePath=%s", filePath)
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

func TestStatFile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		dir      bool
		fileType string
		content  []byte
	}{
		{name: "qdoc", path: "note.qdoc", fileType: "qdoc", content: []byte("{}")},
		{name: "qsheet", path: "budget.qsheet", fileType: "qsheet", content: []byte("{}")},
		{name: "image", path: "photo.jpg", fileType: "image", content: []byte("\xFF\xD8")},
		{name: "video", path: "clip.mp4", fileType: "video", content: []byte("ftyp")},
		{name: "plain directory", path: "photos", dir: true, fileType: "folder"},
		{name: "folder named like qdoc", path: "things.qdoc", dir: true, fileType: "folder"},
		{name: "folder named like qsheet", path: "data.qsheet", dir: true, fileType: "folder"},
		{name: "folder named like image", path: "photo.jpg", dir: true, fileType: "folder"},
		{name: "folder named like video", path: "clip.mp4", dir: true, fileType: "folder"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, filesDir := newTestEngine(t)
			target := filepath.Join(filesDir, tt.path)
			var err error
			if tt.dir {
				err = os.Mkdir(target, 0755)
			} else {
				err = os.WriteFile(target, tt.content, 0644)
			}
			if err != nil {
				t.Fatal(err)
			}
			body, code := statPath(t, engine, tt.path)
			if code != http.StatusOK {
				t.Fatalf("expected 200, got %d", code)
			}
			if isDir, _ := body["isDir"].(bool); isDir != tt.dir {
				t.Errorf("isDir = %v, want %v", isDir, tt.dir)
			}
			if ft, _ := body["fileType"].(string); ft != tt.fileType {
				t.Errorf("fileType = %q, want %q", ft, tt.fileType)
			}
		})
	}
}

// --- stat endpoint: nested paths ---

func TestStatFile_NestedFile(t *testing.T) {
	engine, filesDir := newTestEngine(t)
	sub := filepath.Join(filesDir, "docs")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "report.qdoc"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	body, code := statPath(t, engine, "docs/report.qdoc")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if isDir, _ := body["isDir"].(bool); isDir {
		t.Error("expected isDir=false for a nested qdoc file")
	}
	if ft, _ := body["fileType"].(string); ft != "qdoc" {
		t.Errorf("expected fileType %q, got %q", "qdoc", ft)
	}
}

func TestStatFile_NestedFolderNamedLikeFile(t *testing.T) {
	engine, filesDir := newTestEngine(t)
	sub := filepath.Join(filesDir, "projects")
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

	_, code := statPath(t, engine, "nonexistent.qdoc")
	if code != http.StatusNotFound {
		t.Errorf("expected 404 for missing path, got %d", code)
	}
}
