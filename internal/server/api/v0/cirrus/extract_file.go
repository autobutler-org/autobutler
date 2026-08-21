package v0_files

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
	"github.com/autobutler-org/quark/pkg/vfs"

	"github.com/gin-gonic/gin"
)

// extractFile godoc
// @Summary Extract a zip archive in place
// @Description Extracts a zip file into a subdirectory named after the archive (without its extension) in the same directory
// @Tags cirrus
// @Produce json
// @Param filePath query string true "Path to the zip file to extract"
// @Param serial query string false "Device serial number"
// @Success 200 {object} serverutil.Response "OK"
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /cirrus/extract [post]
func extractFile(c *gin.Context) *serverutil.Response {
	filePath := c.Query("filePath")
	serial := c.Query("serial")

	if filePath == "" {
		return serverutil.BadRequest(errors.New("filePath query parameter is required"))
	}

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	// VFS path: only for .zip archives when no serial is provided.
	// Non-zip types (rar/tar/7z/gz) require mholt/archiver OS-path handling,
	// so those always fall through to StorageService.
	if serial == "" && strings.ToLower(filepath.Ext(filePath)) == ".zip" {
		if reg := deps.VFSRegistry(); reg != nil {
			if fsys, ok := reg.Get("files"); ok {
				if err := extractFileVFS(c, fsys, filePath); err != nil {
					msg := err.Error()
					if strings.Contains(msg, "file not found") {
						return serverutil.NotFound(err)
					}
					return serverutil.InternalServerError(err)
				}
				return serverutil.Ok()
			}
		}
	}

	if _, err := deps.StorageService().ExtractFile(storageutil.ExtractFileParams{
		FilePath:     filePath,
		DeviceSerial: serial,
	}); err != nil {
		msg := err.Error()
		if strings.Contains(msg, "file not found") {
			return serverutil.NotFound(err)
		}
		if strings.Contains(msg, "file is not an archive") || strings.Contains(msg, "only zip archives are supported") {
			return serverutil.BadRequest(err)
		}
		return serverutil.InternalServerError(err)
	}

	return serverutil.Ok()
}

// extractFileVFS extracts a .zip archive via the VFS layer.
// It reads the archive into memory, then writes each entry back via VFS.Write / VFS.MkdirAll.
func extractFileVFS(c *gin.Context, fsys vfs.VFS, filePath string) error {
	ctx := c.Request.Context()

	r, err := fsys.Open(ctx, filePath)
	if err != nil {
		return fmt.Errorf("file not found: %s", filePath)
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read archive: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("failed to open zip archive: %w", err)
	}

	// Determine the destination directory: sibling dir named after the archive stem.
	base := filepath.Base(filePath)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	destDir := path.Join(path.Dir(filePath), stem)

	if err := fsys.MkdirAll(ctx, destDir); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Canonical Zip Slip anchor: every resolved path must begin with this prefix.
	cleanDestDir := path.Clean(destDir) + "/"

	var entryCount int
	for _, f := range zr.File {
		entryCount++
		if entryCount > storageutil.MaxArchiveEntries {
			return fmt.Errorf("archive exceeds maximum of %d entries", storageutil.MaxArchiveEntries)
		}

		// Normalize to forward-slash, clean, and strip any leading slash.
		entryName := strings.TrimPrefix(path.Clean("/"+filepath.ToSlash(f.Name)), "/")
		if entryName == "" || entryName == "." {
			continue
		}

		destPath := path.Join(destDir, entryName)

		// Zip Slip guard: the resolved destination must stay within destDir.
		// This is the canonical check CodeQL and other scanners understand.
		if !strings.HasPrefix(path.Clean(destPath)+"/", cleanDestDir) {
			continue // path traversal attempt — discard silently
		}

		if f.FileInfo().IsDir() {
			if err := fsys.MkdirAll(ctx, destPath); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}
			continue
		}

		// Ensure parent directory exists.
		parentDir := path.Dir(destPath)
		if err := fsys.MkdirAll(ctx, parentDir); err != nil {
			return fmt.Errorf("failed to create parent directory for %s: %w", destPath, err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open archive entry %s: %w", f.Name, err)
		}

		// Apply per-entry size limit.
		limited := io.LimitReader(rc, storageutil.MaxArchiveEntryBytes+1)
		if err := fsys.Write(ctx, destPath, limited, vfs.WriteOptions{}); err != nil {
			rc.Close()
			return fmt.Errorf("failed to write %s: %w", destPath, err)
		}
		rc.Close()
	}

	return nil
}

var extractFileRoute = serverutil.ApiRoute(
	"POST", "/cirrus/extract", extractFile,
)
