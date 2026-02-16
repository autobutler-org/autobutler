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
// @Param baseUpdateURL query string false "Base URL for updates"
// @Success 200 {object} UpdateRequest
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /version/latest [post]
func updateToLatest(c *gin.Context) *serverutil.Response {
	latestVersion, err := updateutil.GetLatestVersion(org, repo)
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	if err := updateutil.Update(updateutil.UpdateParams{
		Version:       latestVersion.TagName,
		BaseUpdateURL: c.Query("baseUpdateURL"),
	}); err != nil {
		return serverutil.InternalServerError(err)
	}

	restartAutobutler()
	return serverutil.Ok().WithData(UpdateRequest{
		Version: latestVersion.TagName,
	})
}

var updateToLatestRoute = serverutil.ApiRoute(
	"POST", "/version/latest", updateToLatest,
)
