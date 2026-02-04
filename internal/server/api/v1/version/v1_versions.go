package v1_version

import "autobutler/pkg/util/serverutil"

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		doUpdateRoute,
		getInstalledVersionRoute,
		listVersionsRoute,
	}
}
