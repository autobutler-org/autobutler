package v1_version

import (
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/updateutil"

	"github.com/gin-gonic/gin"
)

type updateLatestParams struct {
	Source *updateutil.UpdateSource `json:"source" form:"source"`
}

// updateToLatest godoc
// @Summary Update to the latest version
// @Description Finds and installs the latest version
// @Tags version
// @Produce json
// @Param update body updateLatestParams false "Update Request"
// @Success 200 {object} UpdateRequest
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /version/latest [post]
func updateToLatest(c *gin.Context) *serverutil.Response {
	params := updateLatestParams{}
	source := &updateutil.UpdateSource{}
	if err := c.ShouldBind(&params); err == nil {
		source = params.Source
	}

	latestVersion, err := updateutil.GetLatestVersion(source)
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	if err := updateutil.Update(source, latestVersion.Version); err != nil {
		return serverutil.InternalServerError(err)
	}

	restartAutobutler()
	return serverutil.Ok().WithData(UpdateRequest{
		Version: latestVersion.Version,
	})
}

var updateToLatestRoute = serverutil.ApiRoute(
	"POST", "/version/latest", updateToLatest,
)
