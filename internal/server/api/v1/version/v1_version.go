package v1_version

import "github.com/autobutler-org/autobutler/pkg/util/serverutil"

const (
	org  = "autobutler-org"
	repo = "autobutler.org"
)

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		doUpdateRoute,
		getInstalledVersionRoute,
		getLatestVersionRoute,
		getSbomRoute,
		listBranchesRoute,
		listVersionsRoute,
		updateToLatestRoute,
	}
}
