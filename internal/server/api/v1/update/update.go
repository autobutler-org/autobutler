package v1_update

import (
	"autobutler/internal/update"
	"autobutler/pkg/ui/components/error_message"
	"autobutler/pkg/ui/components/success_message"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/updateutil"

	"github.com/gin-gonic/gin"
)

var updateRoute = serverutil.ApiRoute(
	"POST", "/update", func(c *gin.Context) *serverutil.Response {
		version := c.PostForm("version")

		err := updateutil.Update(updateutil.UpdateParams{
			Version: version,
		})

		if err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithComponent(error_message.Component(err.Error()))
		}

		go update.RestartAutobutler()
		return serverutil.Ok().WithComponent(success_message.Component("Update successful, Autobutler will restart."))
	},
)
