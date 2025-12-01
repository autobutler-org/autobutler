package v1_storage

import (
	"github.com/gin-gonic/gin"
)

func SetupRoutes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("/storage/devices/status", getDeviceStatuses)
	apiGroup.POST("/storage/managed", initializeManagedDevice)
}
