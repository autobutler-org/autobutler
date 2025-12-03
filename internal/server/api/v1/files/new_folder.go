package v1_files

import (
	"autobutler/pkg/ui/components/error_message"
	view_files "autobutler/pkg/ui/views/files"
	"autobutler/pkg/util/fileutil"
	"autobutler/pkg/util/serverutil"
	"os"
	"path/filepath"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

var newFolderRoute = serverutil.ApiRoute(
	"POST", "/folder/files/*folderDir", func(c *gin.Context) *serverutil.Response {
		folderDir := c.Param("folderDir")
		folderName := c.PostForm("folderName")
		rootDir := fileutil.GetFilesDir()
		fullPath := filepath.Join(rootDir, folderDir, folderName)

		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithComponent(error_message.Component(err.Error()))
		}

		// Stay in the current directory instead of navigating into the new folder
		currentDir := folderDir
		// Check if it's an HTMX request targeting just the content
		var component templ.Component
		if c.GetHeader("HX-Request") == "true" {
			component = view_files.GetFileExplorerViewContent(c, currentDir, "")
		} else {
			component = view_files.GetFileExplorer(c, currentDir)
		}
		if err := component.Render(c.Request.Context(), c.Writer); err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithComponent(error_message.Component("Failed to render file explorer: " + err.Error()))
		}
		return serverutil.Ok()
	},
)
