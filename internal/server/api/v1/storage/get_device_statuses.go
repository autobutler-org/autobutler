package v1_storage

import (
	"autobutler/pkg/util/storageutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

func getDeviceStatuses(c *gin.Context) {
	statuses, err := storageutil.GetDeviceStatuses()
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
