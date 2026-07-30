package v0_settings

import (
	"github.com/autobutler-org/autobutler/pkg/util/mdnsutil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// LocalDiscoveryResponse describes the current mDNS advertisement state.
type LocalDiscoveryResponse struct {
	// Advertising is true when mDNS is active and the butler is reachable
	// at autobutler.local on the LAN.
	Advertising bool `json:"advertising"`
	// LocalHostname is the mDNS hostname — always "autobutler.local" when
	// advertising is active; empty otherwise.
	LocalHostname string `json:"localHostname,omitempty"`
	// ServiceType is the DNS-SD service type being advertised.
	ServiceType string `json:"serviceType,omitempty"`
}

// getLocalDiscovery godoc
// @Summary Get mDNS local discovery status
// @Description Returns whether the butler is advertising itself on the LAN via mDNS (DNS-SD), and the local hostname (autobutler.local) clients can use to reach it.
// @Tags settings
// @Produce json
// @Success 200 {object} LocalDiscoveryResponse
// @Router /settings/local-discovery [get]
var getLocalDiscoveryRoute = serverutil.ApiRoute(
	"GET", "/settings/local-discovery", func(c *gin.Context) *serverutil.Response {
		advertising := mdnsutil.IsAdvertising()
		resp := LocalDiscoveryResponse{
			Advertising: advertising,
		}
		if advertising {
			resp.LocalHostname = "autobutler.local"
			resp.ServiceType = mdnsutil.ServiceType
		}
		return serverutil.Ok().WithData(resp)
	},
)
