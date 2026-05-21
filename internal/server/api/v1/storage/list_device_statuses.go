package v1_storage

import (
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

// listDeviceStatuses godoc
// @Summary List storage device statuses
// @Description Returns statuses for all known storage devices
// @Tags storage
// @Produce json
// @Success 200 {object} object
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /storage/devices/status [get]
func listDeviceStatuses(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	statuses, err := deps.StorageService().GetDeviceStatuses()
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	// Overlay display names and roles from DB, keyed by device serial.
	if database := deps.Database(); database != nil {
		ctx := c.Request.Context()

		nameMap := make(map[string]string)
		if names, err := database.Queries.GetAllDeviceNames(ctx); err == nil {
			for _, n := range names {
				nameMap[n.DeviceSerial] = n.DisplayName
			}
		}

		roleMap := make(map[string]string)
		if roles, err := database.Queries.GetAllDeviceRoles(ctx); err == nil {
			for _, r := range roles {
				roleMap[r.DeviceSerial] = r.Role
			}
		}

		for i := range statuses {
			serial := ""
			if statuses[i].UsbInfo != nil {
				serial = statuses[i].UsbInfo.GetSerial()
			}
			if name, ok := nameMap[serial]; ok {
				statuses[i].Name = name
			}
			if role, ok := roleMap[serial]; ok {
				statuses[i].Role = role
			} else {
				statuses[i].Role = "unassigned"
			}
		}
	}

	return serverutil.Ok().WithData(gin.H{
		"devices": statuses,
		"count":   len(statuses),
	})
}

var listDeviceStatusesRoute = serverutil.ApiRoute(
	"GET", "/storage/devices/status", listDeviceStatuses,
)
