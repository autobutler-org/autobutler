package v1_version

import (
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/updateutil"
	"autobutler/pkg/util/versionutil"
	"fmt"

	"github.com/gin-gonic/gin"
)

// listVersions godoc
// @Summary List available versions
// @Description Lists all available versions for update
// @Tags version
// @Produce json
// @Param all query bool false "Include all versions, including old and development versions"
// @Success 200 {array} ReleaseJSON
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /version/available [get]
func listVersions(c *gin.Context) *serverutil.Response {
	all := c.Query("all") == "true"
	result, err := updateutil.ListPossibleUpdates(org, repo, all)
	if err != nil {
		return serverutil.NewResponse().
			WithStatusCode(500).
			WithContentType(serverutil.ContentTypeJSON).
			WithData(map[string]string{
				"error": err.Error(),
			})
	}

	currentVersion := versionutil.GetVersion()

	var releases []ReleaseJSON
	for _, r := range result.Releases {
		releases = append(releases, ReleaseJSON{
			TagName:          r.TagName,
			Name:             r.TagName, // GitHub releases often use tag as name
			HtmlUrl:          fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", org, repo, r.TagName),
			PublishedAt:      "",
			IsCurrentVersion: r.TagName == currentVersion.Semver,
		})
	}

	return serverutil.NewResponse().
		WithStatusCode(200).
		WithContentType(serverutil.ContentTypeJSON).
		WithData(releases)
}

var listVersionsRoute = serverutil.ApiRoute(
	"GET", "/version/available", listVersions,
)
