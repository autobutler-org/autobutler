package v0_version

import "github.com/autobutler-org/quark/pkg/util/serverutil"

const org = "autobutler-org"

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		doUpdateRoute,
		getInstalledVersionRoute,
		getSbomRoute,
		listVersionsRoute,
	}
}
