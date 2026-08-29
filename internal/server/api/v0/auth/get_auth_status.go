package v0_auth

import (
	"github.com/autobutler-org/quark/pkg/util/authutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

// getAuthStatus godoc
// @Summary Check auth setup status
// @Description Returns whether initial setup has been completed
// @Tags auth
// @Produce json
// @Success 200 {object} object
// @Router /auth/status [get]
func getAuthStatus(c *gin.Context) *serverutil.Response {
	deps, ok := getQueries(c)
	if !ok || (*deps).Database() == nil {
		return serverutil.Ok().WithData(gin.H{"setup": false})
	}

	complete, err := authutil.IsSetupComplete(c.Request.Context(), (*deps).Database().Queries)
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	return serverutil.Ok().WithData(gin.H{"setup": complete})
}

var getAuthStatusRoute = serverutil.ApiRoute("GET", "/auth/status", getAuthStatus)
