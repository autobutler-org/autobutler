package v0_settings

import "github.com/autobutler-org/quark/pkg/util/serverutil"

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		getSettingsRoute,
		postSettingsRoute,
		getRemoteAccessRoute,
		enableRemoteAccessRoute,
		disableRemoteAccessRoute,
		getUpdateEgressRoute,
		applyUpdateEgressRoute,
	}
}
