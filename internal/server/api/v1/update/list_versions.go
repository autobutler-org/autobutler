package v1_update

import (
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/updateutil"
	"autobutler/pkg/util/versionutil"

	"github.com/gin-gonic/gin"
)

// ReleaseJSON is the JSON representation of a release for the Vue frontend
type ReleaseJSON struct {
	TagName          string `json:"tagName"`
	Name             string `json:"name"`
	HtmlUrl          string `json:"htmlUrl"`
	PublishedAt      string `json:"publishedAt"`
	IsCurrentVersion bool   `json:"isCurrentVersion"`
}

// VersionJSON is the JSON representation of the current version
type VersionJSON struct {
	Semver    string `json:"semver"`
	GitCommit string `json:"gitCommit"`
	GoVersion string `json:"goVersion"`
	BuildDate string `json:"buildDate"`
}

// getCurrentVersion godoc
// @Summary Get current version
// @Description Retrieves the current version of the application
// @Tags update
// @Produce json
// @Success 200 {object} VersionJSON
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /version [get]
func getCurrentVersion(c *gin.Context) *serverutil.Response {
	version := versionutil.GetVersion()

	return serverutil.NewResponse().
		WithStatusCode(200).
		WithContentType(serverutil.ContentTypeJSON).
		WithData(VersionJSON{
			Semver:    version.Semver,
			GitCommit: version.GitCommit,
			GoVersion: version.GoVersion,
			BuildDate: version.BuildDate,
		})
}

var getCurrentVersionRoute = serverutil.ApiRoute(
	"GET", "/version", getCurrentVersion,
)

// listVersions godoc
// @Summary List available versions
// @Description Lists all available versions for update
// @Tags update
// @Produce json
// @Success 200 {array} ReleaseJSON
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /versions [get]
func listVersions(c *gin.Context) *serverutil.Response {
	result, err := updateutil.ListPossibleUpdates(updateutil.ListPossibleUpdatesParams{})
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
			HtmlUrl:          "https://github.com/autobutler-org/autobutler.org/releases/tag/" + r.TagName,
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
	"GET", "/versions", listVersions,
)
