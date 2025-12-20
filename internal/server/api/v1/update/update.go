package v1_update

import (
	"autobutler/internal/update"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/updateutil"

	"github.com/gin-gonic/gin"
)

var updateRoute = serverutil.ApiRoute(
	"POST", "/update", func(c *gin.Context) *serverutil.Response {
		version := c.PostForm("version")

		if err := updateutil.Update(updateutil.UpdateParams{
			Version: version,
		}); err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithError(err)
		}

		go update.RestartAutobutler()
		return serverutil.Ok()
	},
)
