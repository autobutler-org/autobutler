package v0_version

import (
	"errors"

	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/updateutil"

	"github.com/gin-gonic/gin"
)

type UpdateRequest struct {
	Version string `json:"version"`
}

// doUpdate godoc
// @Summary Perform update
// @Description Installs the version and restarts Quark. A second request while that restart is pending is rejected and does not download again.
// @Tags version
// @Accept json
// @Produce json
// @Param update body UpdateParams true "Update Request"
// @Success 200 {object} UpdateRequest
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 409 {object} serverutil.Response "Conflict — an update is already restarting"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Security BearerAuth
// @Router /version/update [post]
func doUpdate(c *gin.Context) *serverutil.Response {
	params := UpdateParams{}
	if err := c.ShouldBind(&params); err != nil {
		return serverutil.BadRequest(err)
	}
	if params.Version == "" {
		return serverutil.BadRequest(
			errors.New("version parameter is required"),
		)
	}

	if err := updateutil.ApplyUpdate(params.Source, params.Version); err != nil {
		if errors.Is(err, updateutil.ErrUpdateInProgress) {
			return serverutil.Conflict(err)
		}
		return serverutil.InternalServerError(err)
	}

	go updateutil.ExitForRestart()
	return serverutil.Ok().WithData(params)
}

var doUpdateRoute = serverutil.ApiRoute(
	"POST", "/version/update", doUpdate,
)
