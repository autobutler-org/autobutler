package ui

import (
	"autobutler/internal/server/ui/types"
	"autobutler/internal/server/ui/views"

	"github.com/gin-gonic/gin"
)

func SetupObservabilityRoutes(router *gin.Engine) {
	setupObservabilityView(router)
}

func setupObservabilityView(router *gin.Engine) {
	uiRoute(router, "/observability", func(c *gin.Context) {
		if err := views.Observability(types.NewPageState()).Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(500)
			return
		}
		c.Status(200)
	})
}
