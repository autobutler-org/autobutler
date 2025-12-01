package v1_storage

import (
	"autobutler/pkg/api"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

func getDeviceStatusesRoute(group *gin.RouterGroup) {
	serverutil.ApiRoute(group, "GET", "/storage/devices/status", func(c *gin.Context) *api.Response {
		statuses, err := storageutil.GetDeviceStatuses()
		if err != nil {
			return api.NewResponse().
				WithContentType(api.ContentTypeJSON).
				WithStatusCode(http.StatusInternalServerError).
				WithData(gin.H{
					"error":   "Failed to get device statuses",
					"details": err.Error(),
				})
		}

		return api.NewResponse().
			WithContentType(api.ContentTypeJSON).
			WithStatusCode(http.StatusOK).
			WithData(gin.H{
				"devices": statuses,
				"count":   len(statuses),
			})
	})
}
