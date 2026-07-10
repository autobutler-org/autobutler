package v0_storage

import (
	"fmt"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

// backupToDevice godoc
// @Summary Do a one-time backup to a device
// @Description Begin a backup to a device
// @Tags storage
// @Produce json
// @Param backupParams body storageutil.BackupToDeviceParams true "Backup parameters"
// @Success 200 {object} object
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /storage/devices/backup [post]
func backupToDevice(c *gin.Context) *serverutil.Response {
	var params storageutil.BackupToDeviceParams
	if err := c.ShouldBindJSON(&params); err != nil {
		return serverutil.BadRequest(fmt.Errorf("Invalid request body: %w", err))
	}

	if params.SourceDeviceSerial == params.TargetDeviceSerial {
		return serverutil.BadRequest(fmt.Errorf("Source and target device cannot be the same"))
	}

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(fmt.Errorf("dependencies not found in context"))
	}
	channel := deps.Worker().GetBackupToDeviceChannel()
	channel <- params
	return serverutil.Ok()
}

var backupToDeviceRoute = serverutil.ApiRoute(
	"POST", "/storage/devices/backup", backupToDevice,
)
