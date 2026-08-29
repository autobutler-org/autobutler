package v0_files

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/fileutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

// extractFile godoc
// @Summary Extract a zip archive in place
// @Description Extracts a zip file into a subdirectory named after the archive (without its extension) in the same directory
// @Tags files
// @Produce json
// @Param filePath query string true "Path to the zip file to extract"
// @Param serial query string false "Device serial number"
// @Success 200 {object} serverutil.Response "OK"
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /files/extract [post]
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
		if fsys := fileutil.FilesVFS(deps.VFSRegistry()); fsys != nil {
			if err := fileutil.ExtractZipVFS(c.Request.Context(), fsys, filePath); err != nil {
				msg := err.Error()
				if strings.Contains(msg, "file not found") {
					return serverutil.NotFound(err)
				}
				return serverutil.InternalServerError(err)
			}
			return serverutil.Ok()
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

var extractFileRoute = serverutil.ApiRoute(
	"POST", "/files/extract", extractFile,
)
