package v1_update

import (
	"github.com/gin-gonic/gin"
)

func SetupRoutes(group *gin.RouterGroup) {
	updateRoute(group)
	listVersionsRoute(group)
}
