package storageutil

import (
	"os"
	"path/filepath"
	"testing"
)

// makeTestDevice creates a ManagedDevice backed by a real temp directory.
func makeTestDevice(t *testing.T) (ManagedDevice, string) {
	t.Helper()
	dir := t.TempDir()
	filesDir := filepath.Join(dir, "files")
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		t.Fatalf("failed to create files dir: %v", err)
	}
	dev := ManagedDevice{
		Device: Device{
			Name:       "test-device",
			MountPoint: dir,
			IsInternal: true,
		},
		DataDir:  dir,
		FilesDir: filesDir,
	}
	return dev, filesDir
}

func TestStatFileImpl_RegularFile(t *testing.T) {
	dev, filesDir := makeTestDevice(t)
	if err := os.WriteFile(filepath.Join(filesDir, "note.abdoc"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := StatFileImpl(StatFileParams{FilePath: "note.abdoc"}, &dev, filesDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsDir {
		t.Error("expected IsDir=false for a regular file")
	}
	if result.FileType != FileTypeAbdoc {
		t.Errorf("expected FileType %q, got %q", FileTypeAbdoc, result.FileType)
	}
	if result.Name != "note.abdoc" {
		t.Errorf("expected Name %q, got %q", "note.abdoc", result.Name)
	}
}

// A folder whose name ends in .abdoc must be reported as a directory, not as
// an abdoc file. This is the core regression case: extension-only heuristics
// would misidentify "things.abdoc/" as a document.
func TestStatFileImpl_FolderNamedLikeAbdoc(t *testing.T) {
	dev, filesDir := makeTestDevice(t)
	if err := os.Mkdir(filepath.Join(filesDir, "things.abdoc"), 0755); err != nil {
		t.Fatal(err)
	}

	result, err := StatFileImpl(StatFileParams{FilePath: "things.abdoc"}, &dev, filesDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsDir {
		t.Error("expected IsDir=true for a folder named things.abdoc")
	}
	if result.FileType != FileTypeFolder {
		t.Errorf("expected FileType %q, got %q", FileTypeFolder, result.FileType)
	}
}

// A folder named like a spreadsheet file must also be reported as a directory.
func TestStatFileImpl_FolderNamedLikeAbsheet(t *testing.T) {
	dev, filesDir := makeTestDevice(t)
	if err := os.Mkdir(filepath.Join(filesDir, "budget.absheet"), 0755); err != nil {
		t.Fatal(err)
	}

	result, err := StatFileImpl(StatFileParams{FilePath: "budget.absheet"}, &dev, filesDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsDir {
		t.Error("expected IsDir=true for a folder named budget.absheet")
	}
	if result.FileType != FileTypeFolder {
		t.Errorf("expected FileType %q, got %q", FileTypeFolder, result.FileType)
	}
}

// A folder named like an image extension must be reported as a directory.
func TestStatFileImpl_FolderNamedLikeImage(t *testing.T) {
	dev, filesDir := makeTestDevice(t)
	if err := os.Mkdir(filepath.Join(filesDir, "photo.jpg"), 0755); err != nil {
		t.Fatal(err)
	}

	result, err := StatFileImpl(StatFileParams{FilePath: "photo.jpg"}, &dev, filesDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsDir {
		t.Error("expected IsDir=true for a folder named photo.jpg")
	}
	if result.FileType != FileTypeFolder {
		t.Errorf("expected FileType %q, got %q", FileTypeFolder, result.FileType)
	}
}

// An actual image file must be reported as an image.
func TestStatFileImpl_ImageFile(t *testing.T) {
	dev, filesDir := makeTestDevice(t)
	if err := os.WriteFile(filepath.Join(filesDir, "sunset.jpg"), []byte("\xFF\xD8"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := StatFileImpl(StatFileParams{FilePath: "sunset.jpg"}, &dev, filesDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsDir {
		t.Error("expected IsDir=false for an image file")
	}
	if result.FileType != FileTypeImage {
		t.Errorf("expected FileType %q, got %q", FileTypeImage, result.FileType)
	}
}

// An actual spreadsheet file must be reported as absheet.
func TestStatFileImpl_AbsheetFile(t *testing.T) {
	dev, filesDir := makeTestDevice(t)
	if err := os.WriteFile(filepath.Join(filesDir, "data.absheet"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := StatFileImpl(StatFileParams{FilePath: "data.absheet"}, &dev, filesDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsDir {
		t.Error("expected IsDir=false for an absheet file")
	}
	if result.FileType != FileTypeAbsheet {
		t.Errorf("expected FileType %q, got %q", FileTypeAbsheet, result.FileType)
	}
}

// A nested path inside a subdirectory must resolve correctly.
func TestStatFileImpl_NestedFile(t *testing.T) {
	dev, filesDir := makeTestDevice(t)
	subDir := filepath.Join(filesDir, "documents")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "report.abdoc"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := StatFileImpl(StatFileParams{FilePath: "documents/report.abdoc"}, &dev, filesDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsDir {
		t.Error("expected IsDir=false for nested abdoc file")
	}
	if result.FileType != FileTypeAbdoc {
		t.Errorf("expected FileType %q, got %q", FileTypeAbdoc, result.FileType)
	}
}

// A nested folder with a misleading extension must still be identified as a
// directory regardless of where it sits in the tree.
func TestStatFileImpl_NestedFolderNamedLikeFile(t *testing.T) {
	dev, filesDir := makeTestDevice(t)
	subDir := filepath.Join(filesDir, "projects")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(subDir, "archive.zip"), 0755); err != nil {
		t.Fatal(err)
	}

	result, err := StatFileImpl(StatFileParams{FilePath: "projects/archive.zip"}, &dev, filesDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsDir {
		t.Error("expected IsDir=true for a folder named archive.zip inside a subdirectory")
	}
	if result.FileType != FileTypeFolder {
		t.Errorf("expected FileType %q, got %q", FileTypeFolder, result.FileType)
	}
}

// Stat-ing a path that does not exist must return an error.
func TestStatFileImpl_NotFound(t *testing.T) {
	dev, filesDir := makeTestDevice(t)

	_, err := StatFileImpl(StatFileParams{FilePath: "nonexistent.abdoc"}, &dev, filesDir)
	if err == nil {
		t.Error("expected error for non-existent path, got nil")
	}
}

// Stat-ing a plain directory (no misleading extension) must report isDir=true.
func TestStatFileImpl_PlainDirectory(t *testing.T) {
	dev, filesDir := makeTestDevice(t)
	if err := os.Mkdir(filepath.Join(filesDir, "photos"), 0755); err != nil {
		t.Fatal(err)
	}

	result, err := StatFileImpl(StatFileParams{FilePath: "photos"}, &dev, filesDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsDir {
		t.Error("expected IsDir=true for a plain directory")
	}
	if result.FileType != FileTypeFolder {
		t.Errorf("expected FileType %q, got %q", FileTypeFolder, result.FileType)
	}
}

// nil device falls back to the provided defaultFilesDir.
func TestStatFileImpl_NilDevice(t *testing.T) {
	dir := t.TempDir()
	filesDir := filepath.Join(dir, "files")
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filesDir, "doc.abdoc"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := StatFileImpl(StatFileParams{FilePath: "doc.abdoc"}, nil, filesDir)
	if err != nil {
		t.Fatalf("unexpected error with nil device: %v", err)
	}
	if result.IsDir {
		t.Error("expected IsDir=false")
	}
	if result.FileType != FileTypeAbdoc {
		t.Errorf("expected FileType %q, got %q", FileTypeAbdoc, result.FileType)
	}
}
