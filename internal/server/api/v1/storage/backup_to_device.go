package v1_storage

import (
	"autobutler/pkg/util/ctxutil"
	"autobutler/pkg/util/deputil"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"
	"fmt"

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
