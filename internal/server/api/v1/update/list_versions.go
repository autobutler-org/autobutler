package v1_update

import (
	"autobutler/pkg/ui/components/error_message"
	"autobutler/pkg/ui/components/landing_nav"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/updateutil"

	"github.com/gin-gonic/gin"
)

var listVersionsRoute = serverutil.ApiRoute(
	"GET", "/versions", func(c *gin.Context) *serverutil.Response {
		result, err := updateutil.ListPossibleUpdates(updateutil.ListPossibleUpdatesParams{})

		if err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithComponent(error_message.Component(err.Error()))
		}

		if err := landing_nav.VersionDropdown(result.Releases).Render(c.Request.Context(), c.Writer); err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithComponent(error_message.Component(err.Error()))
		}
		return serverutil.Ok()
	},
)
