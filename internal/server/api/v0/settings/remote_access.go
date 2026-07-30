package v0_settings

import (
	"errors"

	"github.com/autobutler-org/autobutler/pkg/util/remoteutil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/settingsutil"
	"github.com/gin-gonic/gin"
)

type RemoteAccessRequest struct {
	AuthKey string `json:"authKey"`
}

type RemoteAccessResponse struct {
	Enabled   bool   `json:"enabled"`
	RemoteURL string `json:"remoteUrl,omitempty"`
	// TailscaleDomain is the *.ts.net hostname for which a valid TLS cert is
	// provisioned via Tailscale's Let's Encrypt integration. Non-empty only when
	// MagicDNS and HTTPS are enabled in the tailnet admin console.
	TailscaleDomain string `json:"tailscaleDomain,omitempty"`
	// TailscaleHTTPSURL is the full https:// URL for browser/app access with a
	// trusted cert. Non-empty only when TailscaleDomain is set.
	TailscaleHTTPSURL string `json:"tailscaleHttpsUrl,omitempty"`
}

// remoteAccessStatus builds the current RemoteAccessResponse.
func remoteAccessStatus() RemoteAccessResponse {
	domain := remoteutil.TailscaleDomain()
	resp := RemoteAccessResponse{
		Enabled:         remoteutil.IsRunning(),
		RemoteURL:       remoteutil.RemoteURL(),
		TailscaleDomain: domain,
	}
	if domain != "" {
		resp.TailscaleHTTPSURL = "https://" + domain
	}
	return resp
}

// getRemoteAccess godoc
// @Summary Get remote access status
// @Description Returns whether Tailscale remote access is enabled and the remote URL if available
// @Tags settings
// @Produce json
// @Success 200 {object} RemoteAccessResponse
// @Router /settings/remote-access [get]
var getRemoteAccessRoute = serverutil.ApiRoute(
	"GET", "/settings/remote-access", func(c *gin.Context) *serverutil.Response {
		return serverutil.Ok().WithData(remoteAccessStatus())
	},
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
			return serverutil.Ok().WithData(remoteAccessStatus())
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
		return serverutil.Ok().WithData(remoteAccessStatus())
	},
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
