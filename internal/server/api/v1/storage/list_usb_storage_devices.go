package v1_storage

import (
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/usbutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

var listUsbStorageDevicesRoute = serverutil.ApiRoute(
	"GET", "/storage/devices/usb", func(c *gin.Context) *serverutil.Response {
		usbDevices, err := usbutil.ListUsbDevices(true)
		if err != nil {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusInternalServerError).
				WithData(gin.H{
					"error":   "Failed to list USB devices",
					"details": err.Error(),
				})
		}

		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusOK).
			WithData(gin.H{
				"devices": usbDevices,
				"count":   len(usbDevices),
			})
	},
)
