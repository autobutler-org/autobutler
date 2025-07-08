package ui

import (
	"autobutler/ui/views"

	"github.com/gin-gonic/gin"
)

func SetupIndexRoutes(router *gin.Engine) {
	uiRoute(router, "/", func(c *gin.Context) {
		if err := views.Home().Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
		c.Status(200)
	})
}
