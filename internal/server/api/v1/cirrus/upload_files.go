package v1_files

import (
	"autobutler/pkg/util/cirrusutil"
	"autobutler/pkg/util/ctxutil"
	"autobutler/pkg/util/deputil"
	"autobutler/pkg/util/serverutil"
	"fmt"

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
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		return serverutil.BadRequest(err)
	}

	       form, err := c.MultipartForm()
	       if err != nil {
		       return serverutil.BadRequest(err)
	       }

	       fileHeaders := form.File["files"]
	       returnDir := ""
	       if len(form.Value["returnDir"]) > 0 {
		       returnDir = form.Value["returnDir"][0]
	       }
	       deviceName := c.Query("device")

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(fmt.Errorf("dependencies not found in context"))
	}
	       channel := deps.Worker().GetUploadFilesChannel()
	       channel <- cirrusutil.UploadFilesParams{
		       RootDir:     rootDir,
		       FileHeaders: fileHeaders,
		       ReturnDir:   returnDir,
		       DeviceName:  deviceName,
	       }

	return serverutil.Accepted()
}
