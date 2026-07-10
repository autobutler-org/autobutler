package v0_version

import (
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/updateutil"

	"github.com/gin-gonic/gin"
)

// getLatestVersion godoc
// @Summary Get latest version
// @Description Retrieves the latest available version of the application
// @Tags version
// @Produce json
// @Success 200 {object} updateutil.UpdateVersion "Latest version information"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /version/latest [get]
func getLatestVersion(c *gin.Context) *serverutil.Response {
	version, err := updateutil.GetLatestVersionFromDefaultSources()
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	return serverutil.Ok().WithData(version)
}

var getLatestVersionRoute = serverutil.ApiRoute(
	"GET", "/version/latest", getLatestVersion,
)
