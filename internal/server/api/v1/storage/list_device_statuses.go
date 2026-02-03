package v1_storage

import (
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"
	"net/http"

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
	statuses, err := storageutil.GetDeviceStatuses()
	if err != nil {
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusInternalServerError).
			WithData(gin.H{
				"error":   "Failed to get device statuses",
				"details": err.Error(),
			})
	}

	return serverutil.NewResponse().
		WithContentType(serverutil.ContentTypeJSON).
		WithStatusCode(http.StatusOK).
		WithData(gin.H{
			"devices": statuses,
			"count":   len(statuses),
		})
}

var listDeviceStatusesRoute = serverutil.ApiRoute(
	"GET", "/storage/devices/status", listDeviceStatuses,
)
