package v1_files

import (
	"errors"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

// newFolder godoc
// @Summary Create a new folder
// @Description Enqueue create-folder operation under the given folder directory
// @Tags cirrus
// @Accept multipart/form-data
// @Produce json
// @Param folderDir path string true "Folder directory"
// @Param folderName formData string true "Name of the new folder"
// @Param serial query string false "Device serial number to create folder on"
// @Success 202 {object} serverutil.Response "Ok"
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /cirrus/folder/{folderDir} [post]
func newFolder(c *gin.Context) *serverutil.Response {
	folderDir := c.Param("folderDir")
	folderName := c.PostForm("folderName")
	serial := c.Query("serial")

	if folderDir == "" {
		return serverutil.BadRequest(errors.New("folderDir is required"))
	}
	if folderName == "" {
		return serverutil.BadRequest(errors.New("folderName is required"))
	}

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	if _, err := deps.StorageService().CreateFolder(storageutil.CreateFolderParams{
		FolderDir:    folderDir,
		FolderName:   folderName,
		DeviceSerial: serial,
	}); err != nil {
		return serverutil.InternalServerError(err)
	}

	return serverutil.Ok()
}

var newFolderRoute = serverutil.ApiRoute(
	"POST", "/cirrus/folder/*folderDir", newFolder,
)
