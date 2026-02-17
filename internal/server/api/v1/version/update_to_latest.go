package v1_version

import (
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/updateutil"

	"github.com/gin-gonic/gin"
)

// updateToLatest godoc
// @Summary Update to the latest version
// @Description Finds and installs the latest version
// @Tags version
// @Produce json
// @Success 200 {object} UpdateRequest
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /version/latest [post]
func updateToLatest(c *gin.Context) *serverutil.Response {
	latestVersion, err := updateutil.GetLatestVersionFromDefaultSources()
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	if err := updateutil.UpdateFromDefaultSources(latestVersion); err != nil {
		return serverutil.InternalServerError(err)
	}

	restartAutobutler()
	return serverutil.Ok().WithData(UpdateRequest{
		Version: latestVersion,
	})
}

var updateToLatestRoute = serverutil.ApiRoute(
	"POST", "/version/latest", updateToLatest,
)
