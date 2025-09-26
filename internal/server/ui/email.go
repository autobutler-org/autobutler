package ui

import (
	"autobutler/internal/server/ui/types"
	"autobutler/internal/server/ui/views"

	"github.com/gin-gonic/gin"
)

func SetupEmailRoutes(router *gin.Engine) {
	setupEmailView(router)
}

func setupEmailView(router *gin.Engine) {
	uiRoute(router, "/email", func(c *gin.Context) {
		if err := views.Email(types.NewPageState()).Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
		c.Status(200)
	})
}
