package v1_files

import (
	"autobutler/pkg/util/cirrusutil"
	"autobutler/pkg/util/ctxutil"
	"autobutler/pkg/util/deputil"
	"autobutler/pkg/util/serverutil"
	"fmt"

	"github.com/gin-gonic/gin"
)

type moveFileRequest struct {
	FilePath    string `json:"filePath"`
	NewFilePath string `json:"newFilePath"`
}

var moveFileRoute = serverutil.ApiRoute(
	"PUT", "/cirrus/*filePath", func(c *gin.Context) *serverutil.Response {
		var req moveFileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.BadRequest(err)
		}
		oldDeviceName := c.Query("oldDevice")
		newDeviceName := c.Query("newDevice")

		deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
		if !ok {
			return serverutil.InternalServerError(fmt.Errorf("dependencies not found in context"))
		}
		channel := deps.Worker().GetMoveFileChannel()
		channel <- cirrusutil.MoveFileParams{
			FilePath:      req.FilePath,
			NewFilePath:   req.NewFilePath,
			OldDeviceName: oldDeviceName,
			NewDeviceName: newDeviceName,
		}
		return serverutil.Accepted()
	},
)
