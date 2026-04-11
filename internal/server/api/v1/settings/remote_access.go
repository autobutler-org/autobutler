package v1_settings

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/autobutler-org/autobutler/pkg/util/provisionutil"
	"github.com/autobutler-org/autobutler/pkg/util/remoteutil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/settingsutil"
	"github.com/gin-gonic/gin"
)

// enableMu guards against concurrent calls to enableRemoteAccess, which would
// mint multiple Headscale keys. Only one enable is allowed at a time.
var enableMu sync.Mutex

type RemoteAccessResponse struct {
	Enabled   bool   `json:"enabled"`
	RemoteURL string `json:"remoteUrl,omitempty"`
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
		return serverutil.Ok().WithData(RemoteAccessResponse{
			Enabled:   remoteutil.IsRunning(),
			RemoteURL: remoteutil.RemoteURL(),
		})
	},
)

// enableRemoteAccess godoc
// @Summary Enable remote access via Tailscale
// @Description Auto-provisions a Headscale pre-auth key and starts a tsnet node proxying traffic to the local server
// @Tags settings
// @Accept json
// @Produce json
// @Success 200 {object} RemoteAccessResponse
// @Failure 409 {object} serverutil.Response "Already enabling"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /settings/remote-access [post]
var enableRemoteAccessRoute = serverutil.ApiRoute(
	"POST", "/settings/remote-access", func(c *gin.Context) *serverutil.Response {
		// Guard against concurrent enable calls minting multiple auth keys.
		if !enableMu.TryLock() {
			return serverutil.NewResponse().
				WithStatusCode(http.StatusConflict).
				WithError(fmt.Errorf("remote access enable already in progress"))
		}
		defer enableMu.Unlock()

		if remoteutil.IsRunning() {
			return serverutil.Ok().WithData(RemoteAccessResponse{
				Enabled:   true,
				RemoteURL: remoteutil.RemoteURL(),
			})
		}

		deviceID, err := provisionutil.GetDeviceID()
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("device id: %w", err))
		}

		authKey, err := provisionutil.ProvisionAuthKey(deviceID)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("provision: %w", err))
		}

		if err := remoteutil.Start(authKey); err != nil {
			return serverutil.InternalServerError(err)
		}
		if err := remoteutil.StartProxy(serverutil.ServerPort()); err != nil {
			remoteutil.Stop()
			return serverutil.InternalServerError(err)
		}
		if err := settingsutil.SetRemoteAccess(true); err != nil {
			return serverutil.InternalServerError(err)
		}
		return serverutil.Ok().WithData(RemoteAccessResponse{
			Enabled:   true,
			RemoteURL: remoteutil.RemoteURL(),
		})
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
		if err := settingsutil.SetRemoteAccess(false); err != nil {
			return serverutil.InternalServerError(err)
		}
		return serverutil.Ok().WithData(RemoteAccessResponse{
			Enabled: false,
		})
	},
)
