package v1_version

import (
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/updateutil"
	"fmt"

	"github.com/gin-gonic/gin"
)

// getLatestVersion godoc
// @Summary Get latest version
// @Description Retrieves the latest available version of the application
// @Tags version
// @Produce json
// @Success 200 {object} ReleaseJSON
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /version/latest [get]
func getLatestVersion(c *gin.Context) *serverutil.Response {
	version, err := updateutil.GetLatestVersion(org, repo)
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	return serverutil.Ok().WithData(ReleaseJSON{
		TagName:     version.TagName,
		Name:        version.TagName, // GitHub releases often use tag as name
		HtmlUrl:     fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", org, repo, version.TagName),
		PublishedAt: "",
	})
}

var getLatestVersionRoute = serverutil.ApiRoute(
	"GET", "/version/latest", getLatestVersion,
)
