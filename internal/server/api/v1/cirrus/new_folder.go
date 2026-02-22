package v1_files

import (
	"fmt"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

// createCirrusFolder godoc
// @Summary Create a new folder
// @Description Enqueue create-folder operation under the given folder directory
// @Tags cirrus
// @Accept multipart/form-data
// @Produce json
// @Param folderDir path string true "Folder directory"
// @Param folderName formData string true "Name of the new folder"
// @Param serial query string false "Device serial number to create folder on"
// @Success 202 {object} serverutil.Response "Accepted"
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /cirrus/folder/{folderDir} [post]
func newFolder(c *gin.Context) *serverutil.Response {
	folderDir := c.Param("folderDir")
	folderName := c.PostForm("folderName")
	serial := c.Query("serial")

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(fmt.Errorf("dependencies not found in context"))
	}
	channel := deps.Worker().GetCreateFolderChannel()
	channel <- storageutil.CreateFolderParams{
		FolderDir:    folderDir,
		FolderName:   folderName,
		DeviceSerial: serial,
	}
	return serverutil.Accepted()
}

var newFolderRoute = serverutil.ApiRoute(
	"POST", "/cirrus/folder/*folderDir", newFolder,
)
