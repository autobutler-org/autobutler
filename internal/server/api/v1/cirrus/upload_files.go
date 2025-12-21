package v1_files

import (
	"autobutler/pkg/util/cirrusutil"
	"autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

var uploadRootFilesRoute = serverutil.ApiRoute(
	"POST", "/cirrus", func(c *gin.Context) *serverutil.Response {
		uploadFilesImpl(c, "")
		return serverutil.Ok()
	},
)
var uploadNestedFilesRoutes = serverutil.ApiRoute(
	"POST", "/cirrus/*rootDir", func(c *gin.Context) *serverutil.Response {
		rootDir := c.Param("rootDir")
		uploadFilesImpl(c, rootDir)
		return serverutil.Ok()
	},
)

func uploadFilesImpl(c *gin.Context, rootDir string) {
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		return
	}

	fileHeaders := form.File["files"]
	returnDir := ""
	if len(form.Value["returnDir"]) > 0 {
		returnDir = form.Value["returnDir"][0]
	}

	if _, err := cirrusutil.UploadFiles(cirrusutil.UploadFilesParams{
		RootDir:     rootDir,
		FileHeaders: fileHeaders,
		ReturnDir:   returnDir,
	}); err != nil {
		return
	}
}
