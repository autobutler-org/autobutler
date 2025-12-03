package v1_files

import (
	"autobutler/pkg/ui/components/error_message"
	view_files "autobutler/pkg/ui/views/files"
	"autobutler/pkg/util/fileutil"
	"autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

var moveFileRoute = serverutil.ApiRoute(
	"PUT", "/files/*filePath", func(c *gin.Context) *serverutil.Response {
		filePath := c.Param("filePath")
		newFilePath := c.PostForm("newFilePath")

		result := fileutil.MoveFile(fileutil.MoveFileParams{
			FilePath:    filePath,
			NewFilePath: newFilePath,
		})

		if result.Error != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithComponent(error_message.Component(result.Error.Error()))
		}

		// Always render the full file explorer (JS function targets #file-explorer)
		component := view_files.GetFileExplorer(c, result.NewDir)
		if err := component.Render(c.Request.Context(), c.Writer); err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithComponent(error_message.Component("Failed to render file explorer: " + err.Error()))
		}
		return serverutil.Ok()
	},
)
