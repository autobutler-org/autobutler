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

	return serverutil.Ok().WithData(gin.H{
		"devices": statuses,
		"count":   len(statuses),
	})
}

var listDeviceStatusesRoute = serverutil.ApiRoute(
	"GET", "/storage/devices/status", listDeviceStatuses,
)
