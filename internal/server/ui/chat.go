package ui

import (
	"autobutler/internal/server/ui/views"

	"github.com/gin-gonic/gin"
)

func SetupChatRoutes(router *gin.Engine) {
	uiRoute(router, "/chat", func(c *gin.Context) {
		if err := views.Chat().Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
		c.Status(200)
	})
}
