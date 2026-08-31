package vfs_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/autobutler-org/quark/pkg/vfs"
)

// MemVFS and DBVFS hold content in memory by design. Before #1723 neither
// bounded a write, so an oversized one was an OOM with nothing to diagnose it
// by. It has to come back as ErrTooLarge instead.
func TestMemVFSRefusesAWriteOverTheCap(t *testing.T) {
	fsys := vfs.NewMemVFS("test")
	oversized := io.LimitReader(neverEndingReader{}, vfs.MaxInMemoryWriteBytes+1)

	err := fsys.Write(context.Background(), "big.bin", oversized, vfs.WriteOptions{})
	if !errors.Is(err, vfs.ErrTooLarge) {
		t.Fatalf("want ErrTooLarge, got %v", err)
	}
	if _, statErr := fsys.Stat(context.Background(), "big.bin"); statErr == nil {
		t.Error("a refused write must not leave a truncated entry behind")
	}
}

// An ordinary write is untouched by the cap.
func TestMemVFSAcceptsAnOrdinaryWrite(t *testing.T) {
	fsys := vfs.NewMemVFS("test")

	if err := fsys.Write(context.Background(), "small.txt", strings.NewReader("hello"), vfs.WriteOptions{}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	info, err := fsys.Stat(context.Background(), "small.txt")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Size != 5 {
		t.Errorf("size: got %d, want 5", info.Size)
	}
}

// neverEndingReader supplies bytes without allocating a source of its own, so
// the test can push past a 64 MiB cap without building a 64 MiB fixture.
type neverEndingReader struct{}

func (neverEndingReader) Read(p []byte) (int, error) { return len(p), nil }
