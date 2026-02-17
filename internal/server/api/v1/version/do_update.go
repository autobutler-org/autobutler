package v1_version

import (
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/updateutil"
	"errors"
	"fmt"
	"os"
	"time"

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
// @Param update body UpdateParams true "Update Request"
// @Success 200 {object} UpdateRequest
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
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

	if err := updateutil.Update(params.Source, params.Version); err != nil {
		return serverutil.NewResponse().WithStatusCode(500).WithError(err)
	}

	go restartAutobutler()
	return serverutil.Ok().WithData(params)
}

var doUpdateRoute = serverutil.ApiRoute(
	"POST", "/version/update", doUpdate,
)

func restartAutobutler() {
	fmt.Println("Update complete. Exiting to allow process manager (launchctl/systemd) to restart...")
	time.Sleep(time.Second * 2)
	os.Exit(0)
}
