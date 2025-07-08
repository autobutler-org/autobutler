package v1

import (
	"github.com/gin-gonic/gin"
)

func SetupHealthRoutes(apiV1Group *gin.RouterGroup) {
	healthRoute(apiV1Group)
}

func healthRoute(apiV1Group *gin.RouterGroup) {
	apiRoute(apiV1Group, "GET", "/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})
}
