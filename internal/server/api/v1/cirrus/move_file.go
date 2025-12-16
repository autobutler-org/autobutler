package v1_files

import (
	"autobutler/pkg/ui/components/error_message"
	view_cirrus "autobutler/pkg/ui/views/cirrus"
	"autobutler/pkg/util/cirrusutil"
	"autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

var moveFileRoute = serverutil.ApiRoute(
	"PUT", "/cirrus/*filePath", func(c *gin.Context) *serverutil.Response {
		filePath := c.Param("filePath")
		newFilePath := c.PostForm("newFilePath")

		result, err := cirrusutil.MoveFile(cirrusutil.MoveFileParams{
			FilePath:    filePath,
			NewFilePath: newFilePath,
		})

		if err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithComponent(error_message.Component(err.Error()))
		}

		// Always render the full file explorer (JS function targets #file-explorer)
		component := view_cirrus.GetExplorer(c, result.NewDir)
		if err := component.Render(c.Request.Context(), c.Writer); err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithComponent(error_message.Component("Failed to render file explorer: " + err.Error()))
		}
		return serverutil.Ok()
	},
)
