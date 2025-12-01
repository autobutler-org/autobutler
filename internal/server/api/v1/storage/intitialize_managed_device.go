package v1_storage

import (
	"autobutler/pkg/api"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

func initializeManagedDeviceRoute(group *gin.RouterGroup) {
	serverutil.ApiRoute(group, "POST", "/storage/managed", func(c *gin.Context) *api.Response {
		mountPoint := c.PostForm("mountPoint")

		if mountPoint == "" {
			return api.NewResponse().
				WithContentType(api.ContentTypeJSON).
				WithStatusCode(http.StatusBadRequest).
				WithData(gin.H{
					"error": "Mount point is required",
				})
		}

		err := storageutil.InitializeDeviceDataDir(mountPoint)
		if err != nil {
			return api.NewResponse().
				WithContentType(api.ContentTypeJSON).
				WithStatusCode(http.StatusInternalServerError).
				WithData(gin.H{
					"error":   "Failed to initialize device",
					"details": err.Error(),
				})
		}

		return api.NewResponse().
			WithContentType(api.ContentTypeJSON).
			WithStatusCode(http.StatusOK).
			WithData(gin.H{
				"message":    "Device initialized successfully",
				"mountPoint": mountPoint,
			})
	})
}
