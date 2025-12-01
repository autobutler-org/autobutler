package v1_storage

import (
	"github.com/gin-gonic/gin"
)

func SetupRoutes(group *gin.RouterGroup) {
	getDeviceStatusesRoute(group)
	initializeManagedDeviceRoute(group)
}
