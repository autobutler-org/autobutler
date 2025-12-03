package v1_files

import (
	"autobutler/pkg/ui/components/error_message"
	view_files "autobutler/pkg/ui/views/files"
	"autobutler/pkg/util/fileutil"
	"autobutler/pkg/util/serverutil"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

var moveFileRoute = serverutil.ApiRoute(
	"PUT", "/files/*filePath", func(c *gin.Context) *serverutil.Response {
		filePath := c.Param("filePath")
		newFilePath := c.PostForm("newFilePath")
		filesDir := fileutil.GetFilesDir()
		oldFullPath := filepath.Join(filesDir, filePath)
		newFullPath := filepath.Join(filesDir, newFilePath)

		newFullDir := filepath.Dir(newFullPath)
		if err := os.MkdirAll(newFullDir, 0755); err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithComponent(error_message.Component(err.Error()))
		}
		if err := os.Rename(oldFullPath, newFullPath); err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithComponent(error_message.Component(err.Error()))
		}
		newDir := filepath.Dir(newFilePath)
		if newDir == "." {
			newDir = ""
		}
		// Always render the full file explorer (JS function targets #file-explorer)
		component := view_files.GetFileExplorer(c, newDir)
		if err := component.Render(c.Request.Context(), c.Writer); err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithComponent(error_message.Component("Failed to render file explorer: " + err.Error()))
		}
		return serverutil.Ok()
	},
)
