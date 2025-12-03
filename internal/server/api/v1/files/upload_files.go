package v1_files

import (
	"autobutler/pkg/ui/components/error_message"
	"autobutler/pkg/ui/components/file_explorer/load"
	"autobutler/pkg/ui/types"
	"autobutler/pkg/util/fileutil"
	"autobutler/pkg/util/serverutil"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

var uploadRootFilesRoute = serverutil.ApiRoute(
	"POST", "/files", func(c *gin.Context) *serverutil.Response {
		uploadFilesImpl(c, "")
		return serverutil.Ok()
	},
)
var uploadNestedFilesRoutes = serverutil.ApiRoute(
	"POST", "/files/*rootDir", func(c *gin.Context) *serverutil.Response {
		rootDir := c.Param("rootDir")
		uploadFilesImpl(c, rootDir)
		return serverutil.Ok()
	},
)

func uploadFilesImpl(c *gin.Context, rootDir string) {
	// Parse the multipart form with a max memory size
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
	for _, header := range fileHeaders {
		file, err := header.Open()
		if err != nil {
			error_message.Component("Failed to open file: "+err.Error()).Render(c.Request.Context(), c.Writer)
			return
		}
		defer file.Close()

		fileDir := fileutil.GetFilesDir()
		newFilePath := filepath.Join(fileDir, rootDir, header.Filename)
		if _, err := os.Stat(newFilePath); err == nil {
			ext := filepath.Ext(header.Filename)
			name := header.Filename[:len(header.Filename)-len(ext)]
			i := 1
			for {
				newFileName := fmt.Sprintf("%s_(%d)%s", name, i, ext)
				newFilePath = filepath.Join(fileDir, rootDir, newFileName)
				if _, err := os.Stat(newFilePath); os.IsNotExist(err) {
					break
				}
				i++
			}
		}
		newFile, err := os.Create(newFilePath)
		if err != nil {
			error_message.Component("Failed to create file: "+err.Error()).Render(c.Request.Context(), c.Writer)
			return
		}
		defer newFile.Close()
		if _, err := io.Copy(newFile, file); err != nil {
			error_message.Component("Failed to write file: "+err.Error()).Render(c.Request.Context(), c.Writer)
			return
		}
	}
	returnDir := form.Value["returnDir"]
	if len(returnDir) > 0 {
		rootDir = returnDir[0]
	}
	loadComponent := load.Component(types.NewPageState().WithRootDir(rootDir))
	if err := loadComponent.Render(c.Request.Context(), c.Writer); err != nil {
		c.Status(500)
		return
	}
}
