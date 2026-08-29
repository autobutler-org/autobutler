package v0_settings

import (
	"errors"

	"github.com/autobutler-org/quark/pkg/util/remoteutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/settingsutil"
	"github.com/gin-gonic/gin"
)

// enableRemoteAccess godoc
// @Summary Enable remote access via Tailscale
// @Description Starts a Tailscale tsnet node with the provided auth key and proxies traffic to the local server
// @Tags settings
// @Accept json
// @Produce json
// @Param body body RemoteAccessRequest true "Tailscale auth key"
// @Success 200 {object} RemoteAccessResponse
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /settings/remote-access [post]
var enableRemoteAccessRoute = serverutil.ApiRoute(
	"POST", "/settings/remote-access", func(c *gin.Context) *serverutil.Response {
		if remoteutil.IsRunning() {
			return serverutil.Ok().WithData(RemoteAccessResponse{
				Enabled:   true,
				RemoteURL: remoteutil.RemoteURL(),
			})
		}
		var req RemoteAccessRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.BadRequest(err)
		}
		if req.AuthKey == "" {
			return serverutil.BadRequest(errors.New("authKey is required"))
		}
		if err := remoteutil.Start(req.AuthKey); err != nil {
			return serverutil.InternalServerError(err)
		}
		if err := remoteutil.StartProxy(
			serverutil.ServingPort(),
			serverutil.ServingTLS(),
		); err != nil {
			remoteutil.Stop()
			return serverutil.InternalServerError(err)
		}
		if err := settingsutil.SetRemoteAccess(true, req.AuthKey); err != nil {
			return serverutil.InternalServerError(err)
		}
		return serverutil.Ok().WithData(RemoteAccessResponse{
			Enabled:   true,
			RemoteURL: remoteutil.RemoteURL(),
		})
	},
)
