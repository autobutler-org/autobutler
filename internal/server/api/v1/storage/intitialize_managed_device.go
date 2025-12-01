package v1_storage

import (
	"autobutler/pkg/util/storageutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

func initializeManagedDevice(c *gin.Context) {
	mountPoint := c.PostForm("mountPoint")

	if mountPoint == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Mount point is required",
		})
		return
	}

	err := storageutil.InitializeDeviceDataDir(mountPoint)
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
