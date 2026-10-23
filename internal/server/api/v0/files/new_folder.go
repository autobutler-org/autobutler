package v0_files

import (
	"errors"

	"github.com/autobutler-org/quark/pkg/util/accessutil"
	"github.com/autobutler-org/quark/pkg/util/fileutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

// newFolder godoc
// @Summary Create a new folder
// @Description Enqueue create-folder operation under the given folder directory. Needs write access on that directory; the caller owns the new folder.
// @Tags files
// @Accept multipart/form-data
// @Produce json
// @Param folderDir path string true "Folder directory"
// @Param folderName formData string true "Name of the new folder"
// @Param serial query string false "Device serial number to create folder on"
// @Success 202 {object} serverutil.Response "Ok"
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 403 {object} serverutil.Response "Forbidden"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Security BearerAuth
// @Router /files/folder/{folderDir} [post]
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

	deps, access, resp := loadCaller(c)
	if resp != nil {
		return resp
	}
	if resp := gate(access, serial, folderDir, accessutil.Write); resp != nil {
		return resp
	}

	result, err := fileutil.CreateFolder(fileutil.CreateFolderParams{
		Ctx:        c.Request.Context(),
		Registry:   deps.VFSRegistry(),
		EventBus:   deps.EventBus(),
		FolderDir:  folderDir,
		FolderName: folderName,
		Serial:     serial,
	})
	if err != nil {
		return fileError(err)
	}
	if result.Created {
		accessutil.RecordOwner(c.Request.Context(), deps.Database(), access, serial, result.Path)
	}
	return serverutil.Ok()
}

var newFolderRoute = serverutil.ApiRoute(
	"POST", "/files/folder/*folderDir", newFolder,
)
