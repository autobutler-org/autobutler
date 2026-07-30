package ftsutil_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/autobutler-org/autobutler/pkg/util/ftsutil"
)

func openTestIndex(t *testing.T) *ftsutil.Index {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	idx, err := ftsutil.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { idx.Close() })
	return idx
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return p
}

func TestIndex_SearchFound(t *testing.T) {
	idx := openTestIndex(t)
	dir := t.TempDir()

	absPath := writeFile(t, dir, "notes.txt", "The butler is an AI assistant running on a Raspberry Pi.")

	if err := idx.IndexFile(absPath, "notes.txt"); err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	results, err := idx.Search(context.Background(), "butler", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Path != "notes.txt" {
		t.Errorf("Path: got %q, want %q", results[0].Path, "notes.txt")
	}
}

func TestIndex_SearchNotFound(t *testing.T) {
	idx := openTestIndex(t)
	dir := t.TempDir()
	abs := writeFile(t, dir, "hello.txt", "hello world")
	if err := idx.IndexFile(abs, "hello.txt"); err != nil {
		t.Fatalf("IndexFile: %v", err)
	}
	results, err := idx.Search(context.Background(), "nonexistent", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestIndex_Delete(t *testing.T) {
	idx := openTestIndex(t)
	dir := t.TempDir()
	abs := writeFile(t, dir, "bye.txt", "delete me please")
	if err := idx.IndexFile(abs, "bye.txt"); err != nil {
		t.Fatalf("IndexFile: %v", err)
	}
	if err := idx.Delete("bye.txt"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	results, err := idx.Search(context.Background(), "delete", 10)
	if err != nil {
		t.Fatalf("Search after Delete: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results after delete, got %d", len(results))
	}
}

func TestIndex_Update(t *testing.T) {
	idx := openTestIndex(t)
	dir := t.TempDir()
	abs := writeFile(t, dir, "update.txt", "original content here")
	if err := idx.IndexFile(abs, "update.txt"); err != nil {
		t.Fatalf("IndexFile (initial): %v", err)
	}

	// Overwrite file and re-index.
	if err := os.WriteFile(abs, []byte("completely revised text"), 0644); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	// Force mtime to differ (some filesystems have 1-second granularity).
	if err := idx.Delete("update.txt"); err != nil {
		t.Fatalf("Delete for update: %v", err)
	}
	if err := idx.IndexFile(abs, "update.txt"); err != nil {
		t.Fatalf("IndexFile (update): %v", err)
	}

	// Old term should be gone.
	old, _ := idx.Search(context.Background(), "original", 10)
	if len(old) != 0 {
		t.Errorf("old term still searchable after update, got %d results", len(old))
	}

	// New term should be present.
	new, _ := idx.Search(context.Background(), "revised", 10)
	if len(new) != 1 {
		t.Errorf("new term not found after update, got %d results", len(new))
	}
}

func TestIndex_MultipleFiles(t *testing.T) {
	idx := openTestIndex(t)
	dir := t.TempDir()

	abs1 := writeFile(t, dir, "alpha.txt", "The quick brown fox")
	abs2 := writeFile(t, dir, "beta.md", "The lazy dog jumps")
	abs3 := writeFile(t, dir, "gamma.txt", "The fox and the dog")

	for _, tc := range []struct{ abs, rel string }{
		{abs1, "alpha.txt"}, {abs2, "beta.md"}, {abs3, "gamma.txt"},
	} {
		if err := idx.IndexFile(tc.abs, tc.rel); err != nil {
			t.Fatalf("IndexFile %s: %v", tc.rel, err)
		}
	}

	// "fox" matches alpha and gamma.
	foxResults, err := idx.Search(context.Background(), "fox", 10)
	if err != nil {
		t.Fatalf("Search fox: %v", err)
	}
	if len(foxResults) != 2 {
		t.Errorf("fox: expected 2, got %d", len(foxResults))
	}

	// "dog" matches beta and gamma.
	dogResults, _ := idx.Search(context.Background(), "dog", 10)
	if len(dogResults) != 2 {
		t.Errorf("dog: expected 2, got %d", len(dogResults))
	}
}

func TestIndex_EmptyQuery(t *testing.T) {
	idx := openTestIndex(t)
	results, err := idx.Search(context.Background(), "", 10)
	if err != nil {
		t.Fatalf("Search empty: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty query, got %d", len(results))
	}
}

func TestIndex_Rebuild(t *testing.T) {
	idx := openTestIndex(t)
	dir := t.TempDir()

	writeFile(t, dir, "doc1.txt", "autobutler home server")
	writeFile(t, dir, "doc2.md", "raspberry pi monitoring")
	writeFile(t, dir, "binary.png", "\x89PNG\r\n") // should be skipped

	if err := idx.Rebuild(dir); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	got, _ := idx.Search(context.Background(), "autobutler", 10)
	if len(got) != 1 {
		t.Errorf("expected 1 result for 'autobutler', got %d", len(got))
	}

	// Binary file should not be indexed.
	gotBin, _ := idx.Search(context.Background(), "PNG", 10)
	if len(gotBin) != 0 {
		t.Errorf("expected 0 results for binary content, got %d", len(gotBin))
	}
}
