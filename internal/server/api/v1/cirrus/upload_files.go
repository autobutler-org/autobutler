package v1_files

import (
	"autobutler/pkg/ui/components/error_message"
	"autobutler/pkg/ui/components/file_explorer/load"
	"autobutler/pkg/ui/types"
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
	err := c.Request.ParseMultipartForm(32 << 20)
	if err != nil {
		error_message.Component("Failed to parse multipart form: "+err.Error()).Render(c.Request.Context(), c.Writer)
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		error_message.Component("Failed to get file: "+err.Error()).Render(c.Request.Context(), c.Writer)
		return
	}

	fileHeaders := form.File["files"]
	returnDir := ""
	if len(form.Value["returnDir"]) > 0 {
		returnDir = form.Value["returnDir"][0]
	}

	result, err := cirrusutil.UploadFiles(cirrusutil.UploadFilesParams{
		RootDir:     rootDir,
		FileHeaders: fileHeaders,
		ReturnDir:   returnDir,
	})

	if err != nil {
		error_message.Component(err.Error()).Render(c.Request.Context(), c.Writer)
		return
	}

	loadComponent := load.Component(types.NewPageState().WithRootDir(result.RootDir))
	if err := loadComponent.Render(c.Request.Context(), c.Writer); err != nil {
		c.Status(500)
		return
	}
}
