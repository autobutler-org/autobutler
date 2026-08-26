package v0_files

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

// downloadArchiveFile godoc
// @Summary Download a single file from inside an archive
// @Description Reads the specified entry from the archive and streams it to the client. No data is extracted to disk.
// @Tags files
// @Produce octet-stream
// @Param filePath query string true "Path to the archive file (relative to device files directory)"
// @Param entryPath query string true "Path of the entry inside the archive"
// @Param serial query string false "Device serial number"
// @Success 200 {file} binary "File content"
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 404 {object} serverutil.Response "Entry not found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /files/download-archive-file [get]
func downloadArchiveFile(c *gin.Context) *serverutil.Response {
	archivePath := c.Query("filePath")
	entryPath := c.Query("entryPath")
	serial := c.Query("serial")

	if archivePath == "" || entryPath == "" {
		return serverutil.BadRequest(errors.New("filePath and entryPath query parameters are required"))
	}

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	// VFS path: only when no serial is provided.
	if serial == "" {
		if reg := deps.VFSRegistry(); reg != nil {
			if fsys, ok := reg.Get("files"); ok {
				ctx := c.Request.Context()

				r, err := fsys.Open(ctx, archivePath)
				if err != nil {
					return serverutil.NotFound(err)
				}
				defer r.Close()

				data, err := io.ReadAll(r)
				if err != nil {
					return serverutil.InternalServerError(err)
				}

				zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
				if err != nil {
					return serverutil.InternalServerError(err)
				}

				// Normalize the requested entry path (forward slashes, no leading slash).
				normalizedEntry := strings.Trim(filepath.ToSlash(entryPath), "/")

				for _, f := range zr.File {
					name := strings.Trim(filepath.ToSlash(f.Name), "/")
					if name != normalizedEntry {
						continue
					}

					rc, err := f.Open()
					if err != nil {
						return serverutil.InternalServerError(fmt.Errorf("failed to open archive entry: %w", err))
					}
					defer rc.Close()

					filename := filepath.Base(entryPath)
					size := int64(f.UncompressedSize64)
					c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
					c.Header("Content-Length", strconv.FormatInt(size, 10))
					c.DataFromReader(200, size, "application/octet-stream", rc, nil)
					return nil
				}

				return serverutil.NotFound(fmt.Errorf("entry %q not found in archive", entryPath))
			}
		}
	}

	// StorageService fallback.
	reader, size, err := deps.StorageService().ReadArchiveEntry(storageutil.ReadArchiveEntryParams{
		ArchivePath:  archivePath,
		EntryPath:    entryPath,
		DeviceSerial: serial,
	})
	if err != nil {
		log.Printf("[files] ReadArchiveEntry failed: path=%q entry=%q err=%v", archivePath, entryPath, err)
	}
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	defer reader.Close()

	filename := filepath.Base(entryPath)
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	if size >= 0 {
		c.Header("Content-Length", strconv.FormatInt(size, 10))
	}
	c.DataFromReader(200, size, "application/octet-stream", reader, nil)
	return nil
}

var downloadArchiveFileRoute = serverutil.ApiRoute(
	"GET", "/files/download-archive-file", func(c *gin.Context) *serverutil.Response {
		return downloadArchiveFile(c)
	},
)
