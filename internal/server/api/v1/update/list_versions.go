package v1_update

import (
	"autobutler/internal/update"
	"autobutler/pkg/ui/components/landing_nav"
	"autobutler/pkg/util/serverutil"
	"html"

	"github.com/gin-gonic/gin"
)

var listVersionsRoute = serverutil.ApiRoute(
	"GET", "/versions", func(c *gin.Context) *serverutil.Response {
		releases, err := update.ListPossibleUpdates()
		if err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithData(`<span class="text-red-500">` + html.EscapeString(err.Error()) + `</span>`)
		}
		if err := landing_nav.VersionDropdown(releases).Render(c.Request.Context(), c.Writer); err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithData(`<span class="text-red-500">` + html.EscapeString(err.Error()) + `</span>`)
		}
		return serverutil.Ok()
	},
)
