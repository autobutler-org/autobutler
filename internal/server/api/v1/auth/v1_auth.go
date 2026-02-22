package v1_auth

import "github.com/autobutler-org/autobutler/pkg/util/serverutil"

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		googleAuthorizeRoute,
		googleCallbackRoute,
		googleDisconnectRoute,
	}
}
