package v1_update

import (
	"autobutler/internal/update"
	"autobutler/pkg/api"
	"autobutler/pkg/ui/components/landing_nav"
	"autobutler/pkg/util/serverutil"
	"html"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(group *gin.RouterGroup) {
	updateRoute(group)
	listVersionsRoute(group)
}

func updateRoute(group *gin.RouterGroup) {
	serverutil.ApiRoute(group, "POST", "/update", func(c *gin.Context) *api.Response {
		version := c.PostForm("version")
		if err := update.Update(version); err != nil {
			return api.NewResponse().WithStatusCode(500).WithData(`<span class="text-red-500">` + html.EscapeString(err.Error()) + `</span>`)
		}
		go update.RestartAutobutler()
		return api.Ok().WithData(`<span class="text-green-500">Update successful, Autobutler will restart.</span>`)
	})
}

func listVersionsRoute(group *gin.RouterGroup) {
	serverutil.ApiRoute(group, "GET", "/versions", func(c *gin.Context) *api.Response {
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
