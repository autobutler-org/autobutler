package v0_files

import (
	"errors"
	"path/filepath"
	"strconv"

	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/fileutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"

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

	entry, err := fileutil.OpenArchiveEntry(fileutil.OpenArchiveEntryParams{
		Ctx:         c.Request.Context(),
		Registry:    deps.VFSRegistry(),
		Storage:     deps.StorageService(),
		ArchivePath: archivePath,
		EntryPath:   entryPath,
		Serial:      serial,
	})
	if err != nil {
		return fileError(err)
	}
	defer entry.Reader.Close()

	filename := filepath.Base(entryPath)
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	if entry.Size >= 0 {
		c.Header("Content-Length", strconv.FormatInt(entry.Size, 10))
	}
	c.DataFromReader(200, entry.Size, "application/octet-stream", entry.Reader, nil)
	return nil
}

var downloadArchiveFileRoute = serverutil.ApiRoute(
	"GET", "/files/download-archive-file", func(c *gin.Context) *serverutil.Response {
		return downloadArchiveFile(c)
	},
)
