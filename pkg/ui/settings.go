package ui

import (
	"autobutler/pkg/ui/types"
	"autobutler/pkg/ui/views"
	"autobutler/pkg/util/serverutil"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

func SetupSettingsRoutes(router *gin.Engine) {
	setupSettingsView(router)
	setupThanksView(router)
}

func setupSettingsView(router *gin.Engine) {
	serverutil.UiRoute(router, "/settings", func(c *gin.Context) templ.Component {
		return views.Settings(types.NewPageState())
	})
}

func setupThanksView(router *gin.Engine) {
	serverutil.UiRoute(router, "/thanks", func(c *gin.Context) templ.Component {
		return views.Thanks(types.NewPageState())
	})
}
