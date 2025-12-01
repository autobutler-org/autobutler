package v1_health

import (
	"github.com/gin-gonic/gin"
)

func SetupRoutes(group *gin.RouterGroup) {
	healthRoute(group)
}
