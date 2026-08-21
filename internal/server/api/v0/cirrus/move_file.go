package v0_files

import (
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/eventbus"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/storageutil"

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
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	var req moveFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(err)
	}

	// Use VFS.Move for same-device renames (no serials); fall through to StorageService for cross-device ops.
	if req.OldDeviceSerial == "" && req.NewDeviceSerial == "" {
		if reg := deps.VFSRegistry(); reg != nil {
			if fsys, ok := reg.Get("files"); ok {
				if err := fsys.Move(c.Request.Context(), req.OldFilePath, req.NewFilePath); err != nil {
					return serverutil.InternalServerError(err)
				}
				deps.EventBus().Publish(eventbus.Event{
					Kind:    eventbus.EventMove,
					Path:    req.OldFilePath,
					NewPath: req.NewFilePath,
				})
				return serverutil.Ok()
			}
		}
	}
	if _, err := deps.StorageService().MoveFile(storageutil.MoveFileParams{
		OldFilePath:     req.OldFilePath,
		NewFilePath:     req.NewFilePath,
		OldDeviceSerial: req.OldDeviceSerial,
		NewDeviceSerial: req.NewDeviceSerial,
	}); err != nil {
		return serverutil.InternalServerError(err)
	}

	deps.EventBus().Publish(eventbus.Event{
		Kind:    eventbus.EventMove,
		Path:    req.OldFilePath,
		NewPath: req.NewFilePath,
	})
	return serverutil.Ok()
}

var moveFileRoute = serverutil.ApiRoute(
	"PUT", "/cirrus", moveFile,
)
