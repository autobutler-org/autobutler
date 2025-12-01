package v1_thumbnails

import (
	"github.com/gin-gonic/gin"
)

func SetupRoutes(group *gin.RouterGroup) {
	getThumbnailRoute(group)
}
