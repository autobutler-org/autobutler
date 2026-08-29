package v0_auth

import "github.com/autobutler-org/quark/pkg/util/serverutil"

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		// Local auth
		authStatusRoute,
		authSetupRoute,
		authLoginRoute,
		authLogoutRoute,
		authRecoverRoute,
		// Session management
		listSessionsRoute,
		revokeSessionRoute,
		revokeAllSessionsRoute,
	}
}
