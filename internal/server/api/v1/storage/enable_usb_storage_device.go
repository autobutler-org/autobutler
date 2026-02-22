package v1_storage

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

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
	// Enabling a storage device typically involves mounting it
	serial := c.Param("serial")
	if serial == "" {
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusBadRequest).
			WithError(errors.New("`serial` path parameter is required"))
	}

	targetDevice, err := storageutil.FindUsbDeviceBySerial(serial)
	if err != nil {
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusNotFound).
			WithError(fmt.Errorf("USB device not found: %w", err))
	}

	if !targetDevice.IsStorageDevice() {
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusBadRequest).
			WithError(errors.New("Specified USB device is not a storage device"))
	}

	if mountPath := targetDevice.GetMountPath(); mountPath != "" {
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusBadRequest).
			WithError(errors.New("USB storage device is already mounted"))
	}

	partitions, err := targetDevice.Partitions()
	if err != nil {
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusInternalServerError).
			WithError(fmt.Errorf("Failed to retrieve partitions for USB storage device: %w", err))
	}
	if len(partitions) == 0 {
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusInternalServerError).
			WithError(errors.New("No partitions found on USB storage device"))
	}

	// For simplicity, mount the first partition
	partition := partitions[0]
	mountPath, _ := partition.MountPath()
	if mountPath != "" {
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusBadRequest).
			WithError(errors.New("Partition is already mounted"))
	}

	mountTargetDir := storageutil.GetMountsDir()
	mountTargetPath := filepath.Join(mountTargetDir, targetDevice.GetSerial())
	if err := os.MkdirAll(mountTargetPath, os.ModeDir|os.ModePerm); err != nil {
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusInternalServerError).
			WithError(fmt.Errorf("Failed to create mount target directory: %w", err))
	}
	mountCommand := partition.MountCommand(mountTargetPath)
	if err := mountCommand.Run(); err != nil {
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusInternalServerError).
			WithError(fmt.Errorf("Failed to execute mount command: %w", err))
	}

	return serverutil.NewResponse().
		WithContentType(serverutil.ContentTypeJSON).
		WithStatusCode(http.StatusOK).
		WithData(gin.H{
			"message":    "USB storage device mounted successfully",
			"mount_path": mountTargetPath,
		})
}

var enableUsbStorageDeviceRoute = serverutil.ApiRoute(
	"POST", "/storage/devices/usb/:serial", enableUsbStorageDevice,
)
