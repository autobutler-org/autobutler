package v1_storage

import (
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/usbutil"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

var findUsbStorageDeviceRoute = serverutil.ApiRoute(
	"GET", "/storage/devices/usb/:serial", func(c *gin.Context) *serverutil.Response {
		serial := c.Param("serial")
		if serial == "" {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusBadRequest).
				WithError(errors.New("Serial parameter is required"))
		}
		usbDevice, err := usbutil.FindUsbDeviceBySerial(serial)
		if err != nil {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusInternalServerError).
				WithData(gin.H{
					"error":   "Failed to find USB device",
					"details": err.Error(),
				})
		}

		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusOK).
			WithData(usbDevice)
	},
)
