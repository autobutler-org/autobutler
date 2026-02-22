package v1_version

import (
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/updateutil"

	"github.com/gin-gonic/gin"
)

// listVersions godoc
// @Summary List available versions
// @Description Lists all available versions for update
// @Tags version
// @Produce json
// @Param all query bool false "Include all versions, including old and development versions"
// @Success 200 {array} updateutil.UpdateVersion "List of available versions"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /version/available [get]
func listVersions(c *gin.Context) *serverutil.Response {
	all := c.Query("all") == "true"
	result, err := updateutil.ListPossibleUpdatesFromDefaultSources(all)
	if err != nil {
		return serverutil.NewResponse().
			WithStatusCode(500).
			WithContentType(serverutil.ContentTypeJSON).
			WithData(map[string]string{
				"error": err.Error(),
			})
	}

	return serverutil.NewResponse().
		WithStatusCode(200).
		WithContentType(serverutil.ContentTypeJSON).
		WithData(result.Versions)
}

var listVersionsRoute = serverutil.ApiRoute(
	"GET", "/version/available", listVersions,
)
