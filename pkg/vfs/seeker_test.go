package vfs_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/autobutler-org/autobutler/pkg/vfs"
)

// TestLocalVFS_Open_IsReadSeeker verifies that LocalVFS.Open returns an
// io.ReadSeeker (it wraps *os.File), enabling HTTP range request support in
// the download handler via a simple type assertion.
func TestLocalVFS_Open_IsReadSeeker(t *testing.T) {
	ctx := context.Background()
	v := makeLocalVFS(t, "test")

	content := []byte("hello, seekable world!")
	if err := v.Write(ctx, "file.txt", bytes.NewReader(content), vfs.WriteOptions{}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	rc, err := v.Open(ctx, "file.txt")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer rc.Close()

	rs, ok := rc.(io.ReadSeeker)
	if !ok {
		t.Fatal("LocalVFS.Open did not return an io.ReadSeeker; range requests will not work")
	}

	// Read the whole file.
	all, err := io.ReadAll(rs)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(all) != string(content) {
		t.Errorf("full read: got %q, want %q", all, content)
	}

	// Seek back to start and re-read first 5 bytes.
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
