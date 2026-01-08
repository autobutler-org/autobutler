package v1_storage

import (
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/usbutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

var findUsbStorageDeviceRoute = serverutil.ApiRoute(
	"GET", "/storage/devices/usb/:serial", func(c *gin.Context) *serverutil.Response {
		serial := c.Param("serial")
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
