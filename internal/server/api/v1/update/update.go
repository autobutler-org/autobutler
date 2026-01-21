package v1_update

import (
	"autobutler/internal/update"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/updateutil"
	"errors"

	"github.com/gin-gonic/gin"
)

type UpdateRequest struct {
	Version string `form:"version" binding:"required"`
}

var updateRoute = serverutil.ApiRoute(
	"POST", "/update", func(c *gin.Context) *serverutil.Response {
		updateRequest := UpdateRequest{}
		if err := c.ShouldBind(&updateRequest); err != nil {
			return serverutil.BadRequest(err)
		}
		if updateRequest.Version == "" {
			return serverutil.BadRequest(
				errors.New("version parameter is required"),
			)
		}

		if err := updateutil.Update(updateutil.UpdateParams{
			Version: updateRequest.Version,
		}); err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithError(err)
		}

		go update.RestartAutobutler()
		return serverutil.Ok()
	},
)
