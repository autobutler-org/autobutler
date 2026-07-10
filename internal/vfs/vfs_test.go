package vfs_test

import (
	"testing"

	"github.com/autobutler-org/autobutler/internal/vfs"
)

func makeLocalVFS(t *testing.T, namespaceID string) vfs.VFS {
	t.Helper()
	dir := t.TempDir()
	v, err := vfs.NewLocalVFS(dir, namespaceID)
	if err != nil {
		t.Fatalf("NewLocalVFS: %v", err)
	}
	return v
}

func makeMemVFS(t *testing.T, namespaceID string) vfs.VFS {
	t.Helper()
	return vfs.NewMemVFS(namespaceID)
}

type vfsFactory struct {
	name string
	make func(t *testing.T, ns string) vfs.VFS
}

var factories = []vfsFactory{
	{"LocalVFS", makeLocalVFS},
	{"MemVFS", makeMemVFS},
}

