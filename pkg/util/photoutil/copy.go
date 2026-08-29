package photoutil

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/autobutler-org/quark/pkg/vfs"
)

// CopyPhotoVFS duplicates relPath within the VFS, returning the new relative
// path. The copy lands beside the original as "<name>_copy<ext>", numbered when
// that name is taken. A missing source comes back as [vfs.ErrNotFound].
func CopyPhotoVFS(ctx context.Context, fsys vfs.VFS, relPath string) (string, error) {
	// Verify source exists.
	if _, err := fsys.Stat(ctx, relPath); err != nil {
		return "", err
	}

	ext := filepath.Ext(relPath)
	stem := relPath[:len(relPath)-len(ext)]
	destPath := stem + "_copy" + ext

	// Find a non-conflicting destination name.
	for i := 2; i <= 100; i++ {
		if _, err := fsys.Stat(ctx, destPath); errors.Is(err, vfs.ErrNotFound) {
			break
		}
		destPath = fmt.Sprintf("%s_copy_%d%s", stem, i, ext)
	}

	// Copy: open source, write destination.
	rc, err := fsys.Open(ctx, relPath)
	if err != nil {
		return "", err
	}
	defer rc.Close()

	if err := fsys.Write(ctx, destPath, rc, vfs.WriteOptions{IfNoneMatch: "*"}); err != nil {
		return "", err
	}
	return destPath, nil
}
