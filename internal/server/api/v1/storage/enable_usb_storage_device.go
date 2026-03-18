package v1_storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

// enableUsbStorageDevice godoc
// @Summary Enable (mount) a USB storage device
// @Description Mounts a USB storage device identified by serial and returns mount info
// @Tags storage
// @Produce json
// @Param serial path string true "Device serial"
// @Success 200 {object} object
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /storage/devices/usb/{serial} [post]
func enableUsbStorageDevice(c *gin.Context) *serverutil.Response {
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

	if mountPath := targetDevice.GetMountPath(); mountPath != "" {
		return serverutil.BadRequest(errors.New("USB storage device is already mounted"))
	}

	partitions, err := targetDevice.Partitions()
	if err != nil {
		return serverutil.InternalServerError(fmt.Errorf("failed to retrieve partitions: %w", err))
	}
	if len(partitions) == 0 {
		return serverutil.InternalServerError(errors.New("no partitions found on USB storage device"))
	}

	partition := partitions[0]
	mountPath, _ := partition.MountPath()
	if mountPath != "" {
		return serverutil.BadRequest(errors.New("partition is already mounted"))
	}

	mountTargetDir, err := storageutil.GetMountsDir()
	if err != nil {
		return serverutil.InternalServerError(fmt.Errorf("failed to get mounts directory: %w", err))
	}
	mountTargetPath := filepath.Join(mountTargetDir, targetDevice.GetSerial())
	if err := os.MkdirAll(mountTargetPath, os.ModeDir|os.ModePerm); err != nil {
		return serverutil.InternalServerError(fmt.Errorf("failed to create mount target directory: %w", err))
	}
	mountCommand := partition.MountCommand(mountTargetPath)
	if err := mountCommand.Run(); err != nil {
		return serverutil.InternalServerError(fmt.Errorf("failed to execute mount command: %w", err))
	}

	return serverutil.Ok().WithData(gin.H{
		"message":    "USB storage device mounted successfully",
		"mount_path": mountTargetPath,
	})
}

var enableUsbStorageDeviceRoute = serverutil.ApiRoute(
	"POST", "/storage/devices/usb/:serial", enableUsbStorageDevice,
)
