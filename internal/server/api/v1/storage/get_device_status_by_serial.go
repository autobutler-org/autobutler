package v1_storage

import (
	"autobutler/pkg/util/cirrusutil"
	"autobutler/pkg/util/serverutil"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

var getDeviceStatusBySerialRoute = serverutil.ApiRoute(
	"GET", "/storage/devices/status/:serial", func(c *gin.Context) *serverutil.Response {
		serial := c.Param("serial")
		if serial == "" {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusBadRequest).
				WithError(errors.New("Serial parameter is required"))
		}
		statuses, err := cirrusutil.GetDeviceStatuses()
		if err != nil {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusInternalServerError).
				WithError(fmt.Errorf("Failed to get device statuses: %w", err))
		}
		var device *cirrusutil.DeviceStatus
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
	},
)
