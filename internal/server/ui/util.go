package ui

import (
	"autobutler/pkg/util"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func uiRoute(router *gin.Engine, path string, handler func(c *gin.Context)) gin.IRoutes {
	path = util.TrimLeading(path, '/')
	route := filepath.Join("/", path)
	return router.GET(route, handler)
}
