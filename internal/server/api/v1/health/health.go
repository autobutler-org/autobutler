package v1_health

import (
	"autobutler/pkg/api"
	"autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

func healthRoute(group *gin.RouterGroup) {
	serverutil.ApiRoute(group, "GET", "/health", func(c *gin.Context) *api.Response {
		return api.Ok()
	})
}
