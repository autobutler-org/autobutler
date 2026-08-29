package v0_settings

import (
	"github.com/autobutler-org/quark/pkg/util/serverutil"
)

// SettingsJSON is the JSON representation of application settings.
type SettingsJSON struct {
	AutoUpdate bool `json:"autoUpdate"`
}

type RemoteAccessRequest struct {
	AuthKey string `json:"authKey"`
}

type RemoteAccessResponse struct {
	Enabled   bool   `json:"enabled"`
	RemoteURL string `json:"remoteUrl,omitempty"`
}

type router struct{}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		getSettingsRoute,
		postSettingsRoute,
		getRemoteAccessRoute,
		enableRemoteAccessRoute,
		disableRemoteAccessRoute,
	}
}
