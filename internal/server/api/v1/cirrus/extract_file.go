package v1_files

import (
	"errors"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

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

	if _, err := deps.StorageService().ExtractFile(storageutil.ExtractFileParams{
		FilePath:     filePath,
		DeviceSerial: serial,
	}); err != nil {
		return serverutil.InternalServerError(err)
	}

	return serverutil.Ok()
}

var extractFileRoute = serverutil.ApiRoute(
	"POST", "/cirrus/extract", extractFile,
)
