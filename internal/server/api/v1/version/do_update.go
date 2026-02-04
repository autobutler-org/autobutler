package v1_version

import (
	"autobutler/internal/update"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/updateutil"
	"errors"

	"github.com/gin-gonic/gin"
)

type UpdateRequest struct {
	Version string `json:"version"`
}

// doUpdate godoc
// @Summary Perform update
// @Description Performs an update to the specified version
// @Tags version
// @Accept json
// @Produce json
// @Param update body updateutil.UpdateParams true "Update Request"
// @Success 200 {object} UpdateRequest
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /version/update [post]
func doUpdate(c *gin.Context) *serverutil.Response {
	params := updateutil.UpdateParams{}
	if err := c.ShouldBind(&params); err != nil {
		return serverutil.BadRequest(err)
	}
	if params.Version == "" {
		return serverutil.BadRequest(
			errors.New("version parameter is required"),
		)
	}

	if err := updateutil.Update(params); err != nil {
		return serverutil.NewResponse().WithStatusCode(500).WithError(err)
	}

	go update.RestartAutobutler()
	return serverutil.Ok().WithData(params)
}

var doUpdateRoute = serverutil.ApiRoute(
	"POST", "/version/update", doUpdate,
)
