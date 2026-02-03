package v1_storage

import (
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"
	"errors"
	"fmt"
	"net/http"

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
	serial := c.Param("serial")
	if serial == "" {
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusBadRequest).
			WithError(errors.New("Serial parameter is required"))
	}
	statuses, err := storageutil.GetDeviceStatuses()
	if err != nil {
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusInternalServerError).
			WithError(fmt.Errorf("Failed to get device statuses: %w", err))
	}
	var device *storageutil.DeviceStatus
	for _, ds := range statuses {
		if ds.UsbInfo == nil {
			continue
		}
		if ds.UsbInfo.GetSerial() == serial {
			device = ds
			break
		}
	}
	if device == nil || device.UsbInfo == nil {
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusNotFound).
			WithError(errors.New("Device with specified serial not found"))
	}

	return serverutil.NewResponse().
		WithContentType(serverutil.ContentTypeJSON).
		WithStatusCode(http.StatusOK).
		WithData(device)
}

var getDeviceStatusBySerialRoute = serverutil.ApiRoute(
	"GET", "/storage/devices/status/:serial", getDeviceStatusBySerial,
)
