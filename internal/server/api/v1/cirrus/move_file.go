package v1_files

import (
	"autobutler/pkg/util/cirrusutil"
	"autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

var moveFileRoute = serverutil.ApiRoute(
	"PUT", "/cirrus/*filePath", func(c *gin.Context) *serverutil.Response {
		filePath := c.Param("filePath")
		newFilePath := c.PostForm("newFilePath")

		if _, err := cirrusutil.MoveFile(cirrusutil.MoveFileParams{
			FilePath:    filePath,
			NewFilePath: newFilePath,
		}); err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithError(err)
		}

		return serverutil.Ok()
	},
)
