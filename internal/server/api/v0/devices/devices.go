package v0_devices

import (
	"strconv"
	"time"

	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// ConnectedDeviceJSON is the API representation of a connected device.
type ConnectedDeviceJSON struct {
	ID           int64     `json:"id"`
	IPAddress    string    `json:"ipAddress"`
	UserAgent    string    `json:"userAgent"`
	FirstSeenAt  time.Time `json:"firstSeenAt"`
	LastSeenAt   time.Time `json:"lastSeenAt"`
	RequestCount int64     `json:"requestCount"`
}

// listDevices godoc
// @Summary List connected devices
// @Description Returns all unique client IP + User-Agent combinations that have connected to the butler
// @Tags devices
// @Produce json
// @Success 200 {array} ConnectedDeviceJSON
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /devices [get]
var listDevicesRoute = serverutil.ApiRoute(
	"GET", "/devices", func(c *gin.Context) *serverutil.Response {
		deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
		if !ok {
			return serverutil.InternalServerError(nil)
		}
		rows, err := deps.Database().Queries.ListConnectedDevices(c.Request.Context())
		if err != nil {
			return serverutil.InternalServerError(err)
		}
		result := make([]ConnectedDeviceJSON, len(rows))
		for i, d := range rows {
			result[i] = ConnectedDeviceJSON{
				ID:           d.ID,
				IPAddress:    d.IpAddress,
				UserAgent:    d.UserAgent,
				FirstSeenAt:  d.FirstSeenAt,
				LastSeenAt:   d.LastSeenAt,
				RequestCount: d.RequestCount,
			}
		}
		return serverutil.Ok().WithData(result)
	},
)

// deleteDevice godoc
// @Summary Remove a connected device record
// @Description Deletes a connected device entry by ID
// @Tags devices
// @Param id path int true "Device ID"
// @Success 200 {object} serverutil.Response
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /devices/{id} [delete]
var deleteDeviceRoute = serverutil.ApiRoute(
	"DELETE", "/devices/:id", func(c *gin.Context) *serverutil.Response {
		deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
		if !ok {
			return serverutil.InternalServerError(nil)
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			return serverutil.BadRequest(err)
		}
		if err := deps.Database().Queries.DeleteConnectedDevice(c.Request.Context(), id); err != nil {
			return serverutil.InternalServerError(err)
		}
		return serverutil.Ok()
	},
)
