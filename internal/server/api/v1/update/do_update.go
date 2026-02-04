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

// doUpdate godoc
// @Summary Perform update
// @Description Performs an update to the specified version
// @Tags update
// @Accept json
// @Produce json
// @Param update body UpdateRequest true "Update Request"
// @Success 200 {object} UpdateRequest
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /update [post]
func doUpdate(c *gin.Context) *serverutil.Response {
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
	return serverutil.Ok().WithData(updateRequest)
}

var doUpdateRoute = serverutil.ApiRoute(
	"POST", "/update", doUpdate,
)
