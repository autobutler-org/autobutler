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
	OldFilePath string `json:"oldFilePath"`
	NewFilePath string `json:"newFilePath"`
	OldDevice   string `json:"oldDevice"`
	NewDevice   string `json:"newDevice"`
}

var moveFileRoute = serverutil.ApiRoute(
	"PUT", "/cirrus", func(c *gin.Context) *serverutil.Response {
		var req moveFileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.BadRequest(err)
		}

		deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
		if !ok {
			return serverutil.InternalServerError(fmt.Errorf("dependencies not found in context"))
		}
		channel := deps.Worker().GetMoveFileChannel()
		channel <- cirrusutil.MoveFileParams{
			OldFilePath:   req.OldFilePath,
			NewFilePath:   req.NewFilePath,
			OldDeviceName: req.OldDevice,
			NewDeviceName: req.NewDevice,
		}
		return serverutil.Accepted()
	},
)
