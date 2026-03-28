package v1_settings

import (
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/settingsutil"

	"github.com/gin-gonic/gin"
)

type DevModeResponse struct {
	Enabled bool `json:"enabled"`
}

// getDevMode godoc
// @Summary Get dev mode status
// @Description Returns whether dev mode is enabled
// @Tags settings
// @Produce json
// @Success 200 {object} DevModeResponse
// @Router /settings/dev-mode [get]
func getDevMode(c *gin.Context) *serverutil.Response {
	return serverutil.Ok().WithData(DevModeResponse{
		Enabled: settingsutil.GetDevMode(),
	})
}

var getDevModeRoute = serverutil.ApiRoute(
	"GET", "/settings/dev-mode", getDevMode,
)
