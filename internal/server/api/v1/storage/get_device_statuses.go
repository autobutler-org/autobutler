package v1_storage

import (
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

var getDeviceStatusesRoute = serverutil.ApiRoute(
	"GET", "/storage/devices/status", func(c *gin.Context) *serverutil.Response {
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
	},
)
