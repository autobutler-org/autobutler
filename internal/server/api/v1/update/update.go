package v1_update

import (
	"autobutler/internal/update"
	"autobutler/pkg/util/serverutil"
	"html"

	"github.com/gin-gonic/gin"
)

var updateRoute = serverutil.ApiRoute(
	"POST", "/update", func(c *gin.Context) *serverutil.Response {
		version := c.PostForm("version")
		if err := update.Update(version); err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithData(`<span class="text-red-500">` + html.EscapeString(err.Error()) + `</span>`)
		}
		go update.RestartAutobutler()
		return serverutil.Ok().WithData(`<span class="text-green-500">Update successful, Autobutler will restart.</span>`)
	},
)
