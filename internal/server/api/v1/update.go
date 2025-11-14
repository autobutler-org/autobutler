package v1

import (
	"autobutler/internal/server/ui/components/landing_nav"
	"autobutler/internal/update"
	"fmt"
	"html"

	"github.com/gin-gonic/gin"
)

func SetupUpdateRoutes(apiV1Group *gin.RouterGroup) {
	updateRoute(apiV1Group)
	listVersionsRoute(apiV1Group)
}

func updateRoute(apiV1Group *gin.RouterGroup) {
	apiRoute(apiV1Group, "POST", "/update", func(c *gin.Context) {
		version := c.PostForm("version")
		if err := update.Update(version); err != nil {
			fmt.Fprintf(c.Writer, `<span class="text-red-500">%s</span>`, html.EscapeString(err.Error()))
			return
		}
		go update.RestartAutobutler()
		c.Writer.WriteString(`<span class="text-green-500">Update successful, Autobutler will restart.</span>`)
	})
}

func listVersionsRoute(apiV1Group *gin.RouterGroup) {
	apiRoute(apiV1Group, "GET", "/versions", func(c *gin.Context) {
		releases, err := update.ListPossibleUpdates()
		if err != nil {
			c.Writer.WriteString(`<div class="text-red-500">Failed to fetch versions</div>`)
			return
		}
		if err := landing_nav.VersionDropdown(releases).Render(c.Request.Context(), c.Writer); err != nil {
			c.Writer.WriteString(`<div class="text-red-500">Failed to render versions</div>`)
			return
		}
	})
}
