package v1_storage

import (
	"autobutler/pkg/util/cirrusutil"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/usbutil"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

var enableUsbStorageDeviceRoute = serverutil.ApiRoute(
	"POST", "/storage/devices/usb/:serial", func(c *gin.Context) *serverutil.Response {
		// Enabling a storage device typically involves mounting it
		serial := c.Param("serial")
		if serial == "" {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusBadRequest).
				WithError(errors.New("`serial` path parameter is required"))
		}

		targetDevice, err := usbutil.FindUsbDeviceBySerial(serial)
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

		if _, isMounted := targetDevice.IsMounted(); isMounted {
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

		mountTargetDir := cirrusutil.GetMountsDir()
		mountTargetPath := filepath.Join(mountTargetDir, targetDevice.GetSerial())
		if err := os.MkdirAll(mountTargetPath, os.ModeDir|os.ModePerm); err != nil {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusInternalServerError).
				WithError(fmt.Errorf("Failed to create mount target directory: %w", err))
		}
		mountCommand := partition.MountCommand(mountTargetPath)
		if err != nil {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusInternalServerError).
				WithError(fmt.Errorf("Failed to generate mount command: %w", err))
		}
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
	},
)
