// Package v1_shares implements public share links (issue #1338): authenticated
// management endpoints under /shares, and unauthenticated access endpoints
// under /public/shares/:token. The /api/v1/public/ prefix is exempted from
// session auth in the middleware and rate-limited per IP.
package v1_shares

import "github.com/autobutler-org/autobutler/pkg/util/serverutil"

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		createShareRoute,
		listSharesRoute,
		deleteShareRoute,
		publicShareInfoRoute,
		publicShareDownloadRoute,
	}
}
