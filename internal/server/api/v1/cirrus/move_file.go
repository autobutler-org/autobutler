package v1_files

import (
	"autobutler/pkg/util/cirrusutil"
	"autobutler/pkg/util/ctxutil"
	"autobutler/pkg/util/deputil"
	"autobutler/pkg/util/serverutil"
	"fmt"

	"github.com/gin-gonic/gin"
)

var moveFileRoute = serverutil.ApiRoute(
	"PUT", "/cirrus/*filePath", func(c *gin.Context) *serverutil.Response {
		filePath := c.Param("filePath")
		newFilePath := c.PostForm("newFilePath")

		deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
		if !ok {
			return serverutil.InternalServerError(fmt.Errorf("dependencies not found in context"))
		}
		channel := deps.Worker().GetMoveFileChannel()
		channel <- cirrusutil.MoveFileParams{
			FilePath:    filePath,
			NewFilePath: newFilePath,
		}
		return serverutil.Accepted()
	},
)
