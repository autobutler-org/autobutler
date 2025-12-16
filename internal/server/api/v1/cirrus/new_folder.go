package v1_files

import (
	"autobutler/pkg/ui/components/error_message"
	view_cirrus "autobutler/pkg/ui/views/cirrus"
	"autobutler/pkg/util/fileutil"
	"autobutler/pkg/util/serverutil"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

var newFolderRoute = serverutil.ApiRoute(
	"POST", "/folder/cirrus/*folderDir", func(c *gin.Context) *serverutil.Response {
		folderDir := c.Param("folderDir")
		folderName := c.PostForm("folderName")

		result, err := fileutil.CreateFolder(fileutil.CreateFolderParams{
			FolderDir:  folderDir,
			FolderName: folderName,
		})

		if err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithComponent(error_message.Component(err.Error()))
		}

		// Check if it's an HTMX request targeting just the content
		var component templ.Component
		if c.GetHeader("HX-Request") == "true" {
			component = view_cirrus.GetExplorerViewContent(c, result.CurrentDir, "")
		} else {
			component = view_cirrus.GetExplorer(c, result.CurrentDir)
		}
		if err := component.Render(c.Request.Context(), c.Writer); err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithComponent(error_message.Component("Failed to render file explorer: " + err.Error()))
		}
		return serverutil.Ok()
	},
)
