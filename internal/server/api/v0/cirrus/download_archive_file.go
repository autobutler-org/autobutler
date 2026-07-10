package v0_files

import (
	"errors"
	"log"
	"path/filepath"
	"strconv"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

// downloadArchiveFile godoc
// @Summary Download a single file from inside an archive
// @Description Reads the specified entry from the archive and streams it to the client. No data is extracted to disk.
// @Tags cirrus
// @Produce octet-stream
// @Param filePath query string true "Path to the archive file (relative to device cirrus directory)"
// @Param entryPath query string true "Path of the entry inside the archive"
// @Param serial query string false "Device serial number"
// @Success 200 {file} binary "File content"
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 404 {object} serverutil.Response "Entry not found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /cirrus/download-archive-file [get]
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

	reader, size, err := deps.StorageService().ReadArchiveEntry(storageutil.ReadArchiveEntryParams{
		ArchivePath:  archivePath,
		EntryPath:    entryPath,
		DeviceSerial: serial,
	})
	if err != nil {
		log.Printf("[cirrus] ReadArchiveEntry failed: path=%q entry=%q err=%v", archivePath, entryPath, err)
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
	"GET", "/cirrus/download-archive-file", func(c *gin.Context) *serverutil.Response {
		return downloadArchiveFile(c)
	},
)
