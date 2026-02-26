package v1_files

import (
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

type moveFileRequest struct {
	OldFilePath     string `json:"oldFilePath"`
	NewFilePath     string `json:"newFilePath"`
	OldDeviceSerial string `json:"oldDeviceSerial"`
	NewDeviceSerial string `json:"newDeviceSerial"`
}

// moveFile godoc
// @Summary Move or rename a file
// @Description Enqueue a file move operation between paths/devices
// @Tags cirrus
// @Accept json
// @Produce json
// @Param body body moveFileRequest true "Move file request"
// @Success 202 {object} serverutil.Response "Ok"
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /cirrus [put]
func moveFile(c *gin.Context) *serverutil.Response {
	var req moveFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(err)
	}

	if _, err := storageutil.MoveFile(storageutil.MoveFileParams{
		OldFilePath:     req.OldFilePath,
		NewFilePath:     req.NewFilePath,
		OldDeviceSerial: req.OldDeviceSerial,
		NewDeviceSerial: req.NewDeviceSerial,
	}); err != nil {
		return serverutil.InternalServerError(err)
	}

	return serverutil.Ok()
}

var moveFileRoute = serverutil.ApiRoute(
	"PUT", "/cirrus", moveFile,
)
