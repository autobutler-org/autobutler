package v1_files

import (
	"autobutler/pkg/util/ctxutil"
	"autobutler/pkg/util/deputil"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"
	"fmt"

	"github.com/gin-gonic/gin"
)

// deleteFiles godoc
// @Summary Delete files
// @Description Enqueue deletion of files under the specified root directory
// @Tags cirrus
// @Produce json
// @Param rootDir query string false "Root directory"
// @Param filePaths query []string true "Array of file paths to delete"
// @Param serial query string false "Device serial number to filter by"
// @Success 202 {object} serverutil.Response "Accepted"
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /cirrus [delete]
func deleteFiles(c *gin.Context) *serverutil.Response {
	rootDir := c.Query("rootDir")
	filePaths := c.QueryArray("filePaths")
	serial := c.Query("serial")

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(fmt.Errorf("dependencies not found in context"))
	}
	channel := deps.Worker().GetDeleteFilesChannel()
	channel <- storageutil.DeleteFilesParams{
		RootDir:      rootDir,
		FilePaths:    filePaths,
		DeviceSerial: serial,
	}
	return serverutil.Accepted()
}

var deleteFilesRoute = serverutil.ApiRoute(
	"DELETE", "/cirrus", deleteFiles,
)
