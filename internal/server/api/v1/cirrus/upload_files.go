package v1_files

import (
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

var uploadRootFilesRoute = serverutil.ApiRoute(
	"POST", "/cirrus", func(c *gin.Context) *serverutil.Response {
		return uploadFilesImpl(c, "")
	},
)
var uploadNestedFilesRoutes = serverutil.ApiRoute(
	"POST", "/cirrus/*rootDir", func(c *gin.Context) *serverutil.Response {
		rootDir := c.Param("rootDir")
		return uploadFilesImpl(c, rootDir)
	},
)

func uploadFilesImpl(c *gin.Context, rootDir string) *serverutil.Response {
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
