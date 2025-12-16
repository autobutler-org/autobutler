package v1_storage

import (
	"autobutler/pkg/util/cirrusutil"
	"autobutler/pkg/util/serverutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

var initializeManagedDeviceRoute = serverutil.ApiRoute(
	"POST", "/storage/managed", func(c *gin.Context) *serverutil.Response {
		mountPoint := c.PostForm("mountPoint")

		if mountPoint == "" {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusBadRequest).
				WithData(gin.H{
					"error": "Mount point is required",
				})
		}

		err := cirrusutil.InitializeDeviceDataDir(mountPoint)
		if err != nil {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusInternalServerError).
				WithData(gin.H{
					"error":   "Failed to initialize device",
					"details": err.Error(),
				})
		}

		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusOK).
			WithData(gin.H{
				"message":    "Device initialized successfully",
				"mountPoint": mountPoint,
			})
	},
)
