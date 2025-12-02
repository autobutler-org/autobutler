package v1_files

import (
	view_files "autobutler/pkg/ui/views/files"
	"autobutler/pkg/util/fileutil"
	"autobutler/pkg/util/serverutil"
	"html"
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
			return serverutil.NewResponse().WithStatusCode(500).WithData(`<span class="text-red-500">` + html.EscapeString(err.Error()) + `</span>`)
		}
		if err := os.Rename(oldFullPath, newFullPath); err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithData(`<span class="text-red-500">` + html.EscapeString(err.Error()) + `</span>`)
		}
		newDir := filepath.Dir(newFilePath)
		if newDir == "." {
			newDir = ""
		}
		// Always render the full file explorer (JS function targets #file-explorer)
		component := view_files.GetFileExplorer(c, newDir)
		if err := component.Render(c.Request.Context(), c.Writer); err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithData(`<span class="text-red-500">Failed to render file explorer: ` + html.EscapeString(err.Error()) + `</span>`)
		}
		return serverutil.Ok()
	},
)
