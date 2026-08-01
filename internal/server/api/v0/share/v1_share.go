package v0_share

import "github.com/autobutler-org/autobutler/pkg/util/serverutil"

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		// Authenticated: create, list, revoke
		createShareRoute,
		listSharesRoute,
		revokeShareRoute,
		// Public (no session required): read metadata, download
		getPublicShareRoute,
		downloadPublicShareRoute,
	}
}
