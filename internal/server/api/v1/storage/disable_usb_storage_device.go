package v1_storage

import (
	"errors"
	"fmt"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

// disableUsbStorageDevice godoc
// @Summary Disable (unmount) a USB storage device
// @Description Unmounts a USB storage device identified by serial
// @Tags storage
// @Produce json
// @Param serial path string true "Device serial"
// @Success 200 {object} object
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /storage/devices/usb/{serial} [delete]
func disableUsbStorageDevice(c *gin.Context) *serverutil.Response {
	serial := c.Param("serial")
	if serial == "" {
		return serverutil.BadRequest(errors.New("`serial` path parameter is required"))
	}

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	targetDevice, err := deps.StorageService().FindUsbDeviceBySerial(serial)
	if err != nil {
		return serverutil.NotFound(fmt.Errorf("USB device not found: %w", err))
	}

	if !targetDevice.IsStorageDevice() {
		return serverutil.BadRequest(errors.New("specified USB device is not a storage device"))
	}

	mountPath := targetDevice.GetMountPath()
	if mountPath == "" {
		return serverutil.BadRequest(errors.New("USB storage device is not mounted"))
	}

	unmountCommand := storageutil.UnmountCommand(mountPath)
	if err := unmountCommand.Run(); err != nil {
		return serverutil.InternalServerError(fmt.Errorf("failed to execute unmount command: %w", err))
	}

	return serverutil.Ok().WithData(gin.H{
		"message": "USB storage device unmounted successfully",
	})
}

var disableUsbStorageDeviceRoute = serverutil.ApiRoute(
	"DELETE", "/storage/devices/usb/:serial", disableUsbStorageDevice,
)
