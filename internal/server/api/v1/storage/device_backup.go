package v1_storage

import (
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// setDeviceBackup godoc
// @Summary Mark a device as a backup device
// @Description Persistently mark a USB device (by serial) as a backup device
// @Tags storage
// @Produce json
// @Param serial path string true "Device serial"
// @Success 200 {object} object
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /storage/devices/{serial}/backup [post]
func setDeviceBackup(c *gin.Context) *serverutil.Response {
	serial := c.Param("serial")
	if serial == "" {
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusBadRequest).
			WithError(errors.New("`serial` path parameter is required"))
	}
	if err := storageutil.SetDeviceBackup(serial, true); err != nil {
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusInternalServerError).
			WithError(fmt.Errorf("Failed to persist backup flag: %w", err))
	}
	return serverutil.NewResponse().
		WithContentType(serverutil.ContentTypeJSON).
		WithStatusCode(http.StatusOK).
		WithData(gin.H{"message": "Device marked as backup"})
}

// unsetDeviceBackup godoc
// @Summary Unmark a device as a backup device
// @Description Remove backup marking for a USB device
// @Tags storage
// @Produce json
// @Param serial path string true "Device serial"
// @Success 200 {object} object
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /storage/devices/{serial}/backup [delete]
func unsetDeviceBackup(c *gin.Context) *serverutil.Response {
	serial := c.Param("serial")
	if serial == "" {
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusBadRequest).
			WithError(errors.New("`serial` path parameter is required"))
	}
	if err := storageutil.SetDeviceBackup(serial, false); err != nil {
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusInternalServerError).
			WithError(fmt.Errorf("Failed to persist backup flag: %w", err))
	}
	return serverutil.NewResponse().
		WithContentType(serverutil.ContentTypeJSON).
		WithStatusCode(http.StatusOK).
		WithData(gin.H{"message": "Device unmarked as backup"})
}

var setDeviceBackupRoute = serverutil.ApiRoute(
	"POST", "/storage/devices/:serial/backup", setDeviceBackup,
)

var unsetDeviceBackupRoute = serverutil.ApiRoute(
	"DELETE", "/storage/devices/:serial/backup", unsetDeviceBackup,
)
