package v0_auth

import (
	"github.com/autobutler-org/quark/pkg/util/serverutil"
)

type router struct{}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		// Local auth
		getAuthStatusRoute,
		setupAuthRoute,
		loginUserRoute,
		logoutUserRoute,
		recoverAccountRoute,
		// Session management
		listSessionsRoute,
		revokeSessionRoute,
		revokeAllSessionsRoute,
	}
}
