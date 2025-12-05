package bookutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindAllBooksRecursively(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()

	// Create test files
	testFiles := []struct {
		path    string
		isBook  bool
		content string
	}{
		{"book1.pdf", true, "pdf content"},
		{"book2.epub", true, "epub content"},
		{"readme.txt", false, "text content"},
		{"subdir/book3.pdf", true, "nested pdf"},
		{"subdir/image.jpg", false, "image"},
		{"subdir/nested/book4.epub", true, "deeply nested epub"},
	}

	for _, tf := range testFiles {
		fullPath := filepath.Join(tmpDir, tf.path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(tf.content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Run the function
	books, err := FindAllBooksRecursively(tmpDir)
	if err != nil {
		t.Fatalf("FindAllBooksRecursively failed: %v", err)
	}

	// Count expected books
	expectedCount := 0
	for _, tf := range testFiles {
		if tf.isBook {
			expectedCount++
		}
	}

	// Verify results
	if len(books) != expectedCount {
		t.Errorf("Expected %d books, got %d", expectedCount, len(books))
	}

	// Verify all found files are books
	for _, book := range books {
		ext := filepath.Ext(book.RelPath)
		if ext != ".pdf" && ext != ".epub" {
			t.Errorf("Found non-book file: %s", book.RelPath)
		}
	}
}

func TestFindAllBooksRecursively_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	books, err := FindAllBooksRecursively(tmpDir)
	if err != nil {
		t.Fatalf("FindAllBooksRecursively failed: %v", err)
	}

	if len(books) != 0 {
		t.Errorf("Expected 0 books in empty directory, got %d", len(books))
	}
}

func TestFindAllBooksRecursively_NonExistentDirectory(t *testing.T) {
	_, err := FindAllBooksRecursively("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("Expected error for non-existent directory, got nil")
	}
}

func TestFindAllBooksRecursively_OnlyPDFs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only PDF files
	for i := 1; i <= 3; i++ {
		path := filepath.Join(tmpDir, "book"+string(rune(i))+"pdf.pdf")
		if err := os.WriteFile(path, []byte("pdf"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	books, err := FindAllBooksRecursively(tmpDir)
	if err != nil {
		t.Fatalf("FindAllBooksRecursively failed: %v", err)
	}

	if len(books) != 3 {
		t.Errorf("Expected 3 PDF books, got %d", len(books))
	}
}

func TestFindAllBooksRecursively_OnlyEPUBs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only EPUB files
	for i := 1; i <= 2; i++ {
		path := filepath.Join(tmpDir, "book"+string(rune(i))+"epub.epub")
		if err := os.WriteFile(path, []byte("epub"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	books, err := FindAllBooksRecursively(tmpDir)
	if err != nil {
		t.Fatalf("FindAllBooksRecursively failed: %v", err)
	}

	if len(books) != 2 {
		t.Errorf("Expected 2 EPUB books, got %d", len(books))
	}
}

// Note: The error path in FindAllBooksRecursively where filepath.Rel returns an error
// is extremely difficult to trigger in practice. filepath.Rel only fails when it cannot
// construct a relative path between two paths, which typically requires:
// - Different drive letters on Windows (e.g., C:\ vs D:\)
// - Comparing paths with different volume names
// - Other filesystem boundary issues
// These scenarios cannot be easily simulated in a cross-platform unit test without
// creating filesystem mocks or platform-specific tests. The error handling is present
// for defensive programming but is not covered by tests.

