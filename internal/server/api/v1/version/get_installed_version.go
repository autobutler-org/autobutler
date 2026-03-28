package v1_version

import (
	"os"

	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/versionutil"
	"github.com/gin-gonic/gin"
)

// getInstalledVersion godoc
// @Summary Get installed version
// @Description Retrieves the installed version of the application
// @Tags version
// @Produce json
// @Success 200 {object} VersionJSON
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /version [get]
func getInstalledVersion(c *gin.Context) *serverutil.Response {
	version := versionutil.GetVersion()

	semver := version.Semver
	if branch := os.Getenv("AUTOBUTLER_BRANCH"); branch != "" {
		semver = "dev-" + branch
	}

	return serverutil.Ok().WithData(VersionJSON{
		Semver:    semver,
		GitCommit: version.GitCommit,
		GoVersion: version.GoVersion,
		BuildDate: version.BuildDate,
	})
}

var getInstalledVersionRoute = serverutil.ApiRoute(
	"GET", "/version", getInstalledVersion,
)
