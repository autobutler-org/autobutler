package v1_storage

import (
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/usbutil"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

var disableUsbStorageDeviceRoute = serverutil.ApiRoute(
	"POST", "/storage/devices/usb/:serial/unmount", func(c *gin.Context) *serverutil.Response {
		// Disabling a storage device typically involves unmounting it
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

		mountPath, isMounted := targetDevice.IsMounted()
		if !isMounted {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusBadRequest).
				WithError(errors.New("USB storage device is not mounted"))
		}

		unmountCommand := usbutil.UnmountCommand(mountPath)
		if err != nil {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusInternalServerError).
				WithError(fmt.Errorf("Failed to generate unmount command: %w", err))
		}
		if err := unmountCommand.Run(); err != nil {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusInternalServerError).
				WithError(fmt.Errorf("Failed to execute unmount command: %w", err))
		}

		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusOK).
			WithData(gin.H{
				"message": "USB storage device unmounted successfully",
			})
	},
)
