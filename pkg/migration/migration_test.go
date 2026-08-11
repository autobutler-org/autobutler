package migration_test

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/autobutler-org/autobutler/pkg/migration"
)

// --- helpers -----------------------------------------------------------------

// makeZip builds an in-memory zip archive containing the given name→content
// pairs. Directory entries must have names ending in "/".
func makeZip(t *testing.T, entries map[string]string) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	w := zip.NewWriter(buf)
	for name, content := range entries {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("zip.Create(%q): %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("zip write(%q): %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zip.Close: %v", err)
	}
	return buf
}

// --- ZipExtractor.ListContents -----------------------------------------------

// TestListContents_SingleFile verifies ListContents returns the names of files
// inside a zip archive.
func TestListContents_SingleFile(t *testing.T) {
	z := makeZip(t, map[string]string{"hello.txt": "hello"})
	e := migration.NewZipExtractor()
	contents, err := e.ListContents(context.Background(), z)
	if err != nil {
		t.Fatalf("ListContents: %v", err)
	}
	if len(contents) != 1 {
		t.Fatalf("expected 1 entry, got %d: %v", len(contents), contents)
	}
	if contents[0] != "hello.txt" {
		t.Errorf("contents[0] = %q; want 'hello.txt'", contents[0])
	}
}

// TestListContents_MultipleFiles verifies ListContents returns all file names.
func TestListContents_MultipleFiles(t *testing.T) {
	z := makeZip(t, map[string]string{
		"a.txt": "aaa",
		"b.txt": "bbb",
		"c.txt": "ccc",
	})
	e := migration.NewZipExtractor()
	contents, err := e.ListContents(context.Background(), z)
	if err != nil {
		t.Fatalf("ListContents: %v", err)
	}
	if len(contents) != 3 {
		t.Errorf("expected 3 entries, got %d: %v", len(contents), contents)
	}
}

// TestListContents_Empty verifies ListContents returns an empty slice for an
// empty zip.
func TestListContents_Empty(t *testing.T) {
	z := makeZip(t, map[string]string{})
	e := migration.NewZipExtractor()
	contents, err := e.ListContents(context.Background(), z)
	if err != nil {
		t.Fatalf("ListContents: %v", err)
	}
	if len(contents) != 0 {
		t.Errorf("expected 0 entries, got %d", len(contents))
	}
}

// TestListContents_InvalidZip verifies ListContents returns an error for
// non-zip data.
func TestListContents_InvalidZip(t *testing.T) {
	e := migration.NewZipExtractor()
	_, err := e.ListContents(context.Background(), bytes.NewReader([]byte("not a zip")))
	if err == nil {
		t.Error("expected error for invalid zip, got nil")
	}
}

// --- ZipExtractor.Extract ----------------------------------------------------

// TestExtract_SingleFile verifies Extract unpacks a single file into destDir.
func TestExtract_SingleFile(t *testing.T) {
	z := makeZip(t, map[string]string{"data.txt": "content here"})
	destDir := t.TempDir()
	e := migration.NewZipExtractor()
	if err := e.Extract(context.Background(), z, destDir); err != nil {
		t.Fatalf("Extract: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(destDir, "data.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "content here" {
		t.Errorf("file content = %q; want 'content here'", got)
	}
}

// TestExtract_NestedPath verifies Extract creates parent directories for
// nested paths.
func TestExtract_NestedPath(t *testing.T) {
	z := makeZip(t, map[string]string{"a/b/c.txt": "nested"})
	destDir := t.TempDir()
	e := migration.NewZipExtractor()
	if err := e.Extract(context.Background(), z, destDir); err != nil {
		t.Fatalf("Extract: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(destDir, "a", "b", "c.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "nested" {
		t.Errorf("content = %q; want 'nested'", got)
	}
}

// TestExtract_InvalidZip verifies Extract returns an error for non-zip data.
func TestExtract_InvalidZip(t *testing.T) {
	destDir := t.TempDir()
	e := migration.NewZipExtractor()
	err := e.Extract(context.Background(), bytes.NewReader([]byte("garbage")), destDir)
	if err == nil {
		t.Error("expected error for invalid zip, got nil")
	}
}

// --- InMemoryJobStore --------------------------------------------------------

func newJob(id string) *migration.ImportJob {
	return &migration.ImportJob{
		ID:        id,
		ExportID:  "export-" + id,
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// TestJobStore_CreateAndGet verifies Create followed by Get returns the job.
func TestJobStore_CreateAndGet(t *testing.T) {
	store := migration.NewInMemoryJobStore()
	ctx := context.Background()
	job := newJob("job-1")

	if err := store.Create(ctx, job); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := store.Get(ctx, "job-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "job-1" {
		t.Errorf("ID = %q; want 'job-1'", got.ID)
	}
}

// TestJobStore_GetMissing verifies Get returns an error for an unknown job.
func TestJobStore_GetMissing(t *testing.T) {
	store := migration.NewInMemoryJobStore()
	_, err := store.Get(context.Background(), "ghost")
	if err == nil {
		t.Error("expected error for missing job, got nil")
	}
}

// TestJobStore_CreateDuplicate verifies Create returns an error when the job
// ID already exists.
func TestJobStore_CreateDuplicate(t *testing.T) {
	store := migration.NewInMemoryJobStore()
	ctx := context.Background()
	job := newJob("dup")

	store.Create(ctx, job)
	err := store.Create(ctx, job)
	if err == nil {
		t.Error("expected error for duplicate job ID, got nil")
	}
}

// TestJobStore_Update verifies Update persists changes to an existing job.
func TestJobStore_Update(t *testing.T) {
	store := migration.NewInMemoryJobStore()
	ctx := context.Background()
	job := newJob("upd")
	store.Create(ctx, job)

	job.Status = "completed"
	job.Progress = 1.0
	if err := store.Update(ctx, job); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := store.Get(ctx, "upd")
	if got.Status != "completed" {
		t.Errorf("Status = %q; want 'completed'", got.Status)
	}
	if got.Progress != 1.0 {
		t.Errorf("Progress = %v; want 1.0", got.Progress)
	}
}

// TestJobStore_UpdateMissing verifies Update returns an error for a job that
// doesn't exist.
func TestJobStore_UpdateMissing(t *testing.T) {
	store := migration.NewInMemoryJobStore()
	err := store.Update(context.Background(), newJob("ghost"))
	if err == nil {
		t.Error("expected error updating missing job, got nil")
	}
}

// TestJobStore_List verifies List returns all created jobs.
func TestJobStore_List(t *testing.T) {
	store := migration.NewInMemoryJobStore()
	ctx := context.Background()

	store.Create(ctx, newJob("j1"))
	store.Create(ctx, newJob("j2"))
	store.Create(ctx, newJob("j3"))

	jobs, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(jobs) != 3 {
		t.Errorf("expected 3 jobs, got %d", len(jobs))
	}
}

// TestJobStore_ListEmpty verifies List returns an empty slice when no jobs
// have been created.
func TestJobStore_ListEmpty(t *testing.T) {
	store := migration.NewInMemoryJobStore()
	jobs, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}
}
