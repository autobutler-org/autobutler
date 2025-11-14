package ui

import (
	"autobutler/internal/server/ui/types"
	"autobutler/internal/server/ui/views"
	"autobutler/internal/serverutil"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

func SetupHealthRoutes(router *gin.Engine) {
	setupHealthView(router)
}

func setupHealthView(router *gin.Engine) {
	serverutil.UiRoute(router, "/health", func(c *gin.Context) templ.Component {
		return views.Health(types.NewPageState())
	})
}
