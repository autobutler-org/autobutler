package v1_files

import (
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

// uploadFiles godoc
// @Summary Upload files to the top-level directory
// @Description Upload one or more files via multipart/form-data
// @Tags cirrus
// @Accept multipart/form-data
// @Produce json
// @Param serial query string false "Device serial number to upload to"
// @Param file formData file true "File to upload"
// @Success 200 {object} serverutil.Response "OK"
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Router /cirrus/upload [post]
func uploadFiles(c *gin.Context) *serverutil.Response {
	return uploadFilesNested(c, "")
}

// uploadFiles godoc
// @Summary Upload files to a nested directory
// @Description Upload one or more files via multipart/form-data
// @Tags cirrus
// @Accept multipart/form-data
// @Produce json
// @Param rootDir path string true "Directory to upload into"
// @Param serial query string false "Device serial number to upload to"
// @Param file formData file true "File to upload"
// @Success 200 {object} serverutil.Response "OK"
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Router /cirrus//upload/{rootDir} [post]
func uploadFilesNested(c *gin.Context, rootDir string) *serverutil.Response {
	serial := c.Query("serial")
	reader, err := c.Request.MultipartReader()
	if err != nil {
		return serverutil.BadRequest(err)
	}
	err = storageutil.UploadFilesStreamed(storageutil.UploadFilesStreamedParams{
		Reader:       reader,
		RootDir:      rootDir,
		DeviceSerial: serial,
	})
	if err != nil {
		return serverutil.BadRequest(err)
	}
	return serverutil.Ok()
}

var uploadFilesRoute = serverutil.ApiRoute(
	"POST", "/cirrus/upload", func(c *gin.Context) *serverutil.Response {
		return uploadFiles(c)
	},
)

var uploadFilesNestedRoute = serverutil.ApiRoute(
	"POST", "/cirrus//upload/*rootDir", func(c *gin.Context) *serverutil.Response {
		return uploadFilesNested(c, c.Param("rootDir"))
	},
)
