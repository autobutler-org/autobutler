package v1_storage

import (
	"errors"
	"fmt"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// getDeviceStatusBySerial godoc
// @Summary Get storage device status by serial
// @Description Returns status for a single storage device identified by serial
// @Tags storage
// @Produce json
// @Param serial path string true "Device serial"
// @Success 200 {object} object
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /storage/devices/status/{serial} [get]
func getDeviceStatusBySerial(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	serial := c.Param("serial")
	if serial == "" {
		return serverutil.BadRequest(errors.New("serial parameter is required"))
	}

	statuses, err := deps.StorageService().GetDeviceStatuses()
	if err != nil {
		return serverutil.InternalServerError(fmt.Errorf("failed to get device statuses: %w", err))
	}

	for _, ds := range statuses {
		if ds.UsbInfo != nil && ds.UsbInfo.GetSerial() == serial {
			// Overlay custom display name when available.
			if database := deps.Database(); database != nil {
				if name, err := database.Queries.GetDeviceName(c.Request.Context(), ds.DevicePath); err == nil && name != "" {
					ds.Name = name
				}
			}
			return serverutil.Ok().WithData(ds)
		}
	}

	return serverutil.NotFound(errors.New("device with specified serial not found"))
}

var getDeviceStatusBySerialRoute = serverutil.ApiRoute(
	"GET", "/storage/devices/status/:serial", getDeviceStatusBySerial,
)
