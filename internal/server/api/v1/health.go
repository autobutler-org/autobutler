package v1

import (
	"autobutler/internal/serverutil"
	"autobutler/pkg/api"

	"github.com/gin-gonic/gin"
)

func SetupHealthRoutes(apiV1Group *gin.RouterGroup) {
	healthRoute(apiV1Group)
}

func healthRoute(apiV1Group *gin.RouterGroup) {
	serverutil.ApiRoute(apiV1Group, "GET", "/health", func(c *gin.Context) *api.Response {
		return api.Ok()
	})
}
