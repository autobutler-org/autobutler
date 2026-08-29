package v0_settings

import (
	"github.com/autobutler-org/quark/pkg/util/remoteutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/settingsutil"
	"github.com/gin-gonic/gin"
)

// disableRemoteAccess godoc
// @Summary Disable remote access
// @Description Stops the Tailscale tsnet node
// @Tags settings
// @Produce json
// @Success 200 {object} RemoteAccessResponse
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /settings/remote-access [delete]
var disableRemoteAccessRoute = serverutil.ApiRoute(
	"DELETE", "/settings/remote-access", func(c *gin.Context) *serverutil.Response {
		remoteutil.Stop()
		if err := settingsutil.SetRemoteAccess(false, ""); err != nil {
			return serverutil.InternalServerError(err)
		}
		return serverutil.Ok().WithData(RemoteAccessResponse{
			Enabled: false,
		})
	},
)
