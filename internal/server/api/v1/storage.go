package v1

import (
	"autobutler/pkg/storage"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupStorageRoutes configures storage-related API routes
func SetupStorageRoutes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("/storage/devices/status", getDeviceStatuses)
	apiGroup.POST("/storage/managed", initializeManagedDevice)
}

func initializeManagedDevice(c *gin.Context) {
	mountPoint := c.PostForm("mountPoint")

	if mountPoint == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Mount point is required",
		})
		return
	}

	err := storage.InitializeDeviceDataDir(mountPoint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to initialize device",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Device initialized successfully",
		"mountPoint": mountPoint,
	})
}

func getDeviceStatuses(c *gin.Context) {
	statuses, err := storage.GetDeviceStatuses()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get device statuses",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"devices": statuses,
		"count":   len(statuses),
	})
}
