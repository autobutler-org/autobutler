package v1_update

import (
	"autobutler/internal/update"
	"autobutler/pkg/api"
	"autobutler/pkg/util/serverutil"
	"html"

	"github.com/gin-gonic/gin"
)

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
