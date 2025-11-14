package v1

import (
	"autobutler/internal/server/ui/components/landing_nav"
	"autobutler/internal/serverutil"
	"autobutler/internal/update"
	"autobutler/pkg/api"
	"html"

	"github.com/gin-gonic/gin"
)

func SetupUpdateRoutes(apiV1Group *gin.RouterGroup) {
	updateRoute(apiV1Group)
	listVersionsRoute(apiV1Group)
}

func updateRoute(apiV1Group *gin.RouterGroup) {
	serverutil.ApiRoute(apiV1Group, "POST", "/update", func(c *gin.Context) *api.Response {
		version := c.PostForm("version")
		if err := update.Update(version); err != nil {
			return api.NewResponse().WithStatusCode(500).WithData(`<span class="text-red-500">` + html.EscapeString(err.Error()) + `</span>`)
		}
		go update.RestartAutobutler()
		return api.Ok().WithData(`<span class="text-green-500">Update successful, Autobutler will restart.</span>`)
	})
}

func listVersionsRoute(apiV1Group *gin.RouterGroup) {
	serverutil.ApiRoute(apiV1Group, "GET", "/versions", func(c *gin.Context) *api.Response {
		releases, err := update.ListPossibleUpdates()
		if err != nil {
			return api.NewResponse().WithStatusCode(500).WithData(`<span class="text-red-500">` + html.EscapeString(err.Error()) + `</span>`)
		}
		if err := landing_nav.VersionDropdown(releases).Render(c.Request.Context(), c.Writer); err != nil {
			return api.NewResponse().WithStatusCode(500).WithData(`<span class="text-red-500">` + html.EscapeString(err.Error()) + `</span>`)
		}
		return api.Ok()
	})
}
