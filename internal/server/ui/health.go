package ui

import (
	"autobutler/internal/server/ui/types"
	"autobutler/internal/server/ui/views"

	"github.com/gin-gonic/gin"
)

func SetupHealthRoutes(router *gin.Engine) {
	setupHealthView(router)
}

func setupHealthView(router *gin.Engine) {
	uiRoute(router, "/health", func(c *gin.Context) {
		if err := views.Health(types.NewPageState()).Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(500)
			return
		}
		c.Status(200)
	})
}
