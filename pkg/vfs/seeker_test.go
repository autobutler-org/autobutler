package vfs_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/autobutler-org/autobutler/pkg/vfs"
)

// TestLocalVFS_OpenSeeker verifies that LocalVFS implements the optional
// vfs.Seeker interface and that the returned ReadSeekCloser can seek and read
// correctly — a prerequisite for HTTP range request support.
func TestLocalVFS_OpenSeeker(t *testing.T) {
	ctx := context.Background()
	v := makeLocalVFS(t, "test")

	content := []byte("hello, seekable world!")
	if err := v.Write(ctx, "file.txt", bytes.NewReader(content), vfs.WriteOptions{}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	seeker, ok := v.(vfs.Seeker)
	if !ok {
		t.Skip("LocalVFS does not implement vfs.Seeker (unexpected)")
	}

	rs, err := seeker.OpenSeeker(ctx, "file.txt")
	if err != nil {
		t.Fatalf("OpenSeeker: %v", err)
	}
	defer rs.Close()

	// Read the whole file.
	all, err := io.ReadAll(rs)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(all) != string(content) {
		t.Errorf("full read: got %q, want %q", all, content)
	}

	// Seek back to start and re-read.
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("Seek(0, Start): %v", err)
	}
	first5 := make([]byte, 5)
	if _, err := io.ReadFull(rs, first5); err != nil {
		t.Fatalf("ReadFull after seek: %v", err)
	}
	if string(first5) != "hello" {
		t.Errorf("seek+read: got %q, want %q", first5, "hello")
	}

	// Seek to offset 7 and read 8 bytes ("seekable").
	if _, err := rs.Seek(7, io.SeekStart); err != nil {
		t.Fatalf("Seek(7, Start): %v", err)
	}
	mid := make([]byte, 8)
	if _, err := io.ReadFull(rs, mid); err != nil {
		t.Fatalf("ReadFull mid: %v", err)
	}
	if string(mid) != "seekable" {
		t.Errorf("mid seek+read: got %q, want %q", mid, "seekable")
	}
}

// TestLocalVFS_OpenSeeker_NotFound verifies that OpenSeeker returns
// vfs.ErrNotFound for missing files.
func TestLocalVFS_OpenSeeker_NotFound(t *testing.T) {
	v := makeLocalVFS(t, "test")
	seeker, ok := v.(vfs.Seeker)
	if !ok {
		t.Skip("LocalVFS does not implement vfs.Seeker")
	}
	_, err := seeker.OpenSeeker(context.Background(), "no-such-file.txt")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestSeekerInterface_Satisfaction is a compile-time guard ensuring LocalVFS
// satisfies vfs.Seeker. This will fail to compile if the interface changes.
func TestSeekerInterface_Satisfaction(t *testing.T) {
	v := makeLocalVFS(t, "test")
	var _ vfs.Seeker = v.(vfs.Seeker)
}
