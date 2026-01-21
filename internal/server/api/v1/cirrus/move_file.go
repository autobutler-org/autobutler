package v1_files

import (
	"autobutler/pkg/util/ctxutil"
	"autobutler/pkg/util/deputil"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"
	"fmt"

	"github.com/gin-gonic/gin"
)

type moveFileRequest struct {
	OldFilePath     string `json:"oldFilePath"`
	NewFilePath     string `json:"newFilePath"`
	OldDeviceSerial string `json:"oldDeviceSerial"`
	NewDeviceSerial string `json:"newDeviceSerial"`
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
		channel <- storageutil.MoveFileParams{
			OldFilePath:     req.OldFilePath,
			NewFilePath:     req.NewFilePath,
			OldDeviceSerial: req.OldDeviceSerial,
			NewDeviceSerial: req.NewDeviceSerial,
		}
		return serverutil.Accepted()
	},
)
