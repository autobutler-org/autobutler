package v1_smb

import "github.com/autobutler-org/autobutler/pkg/util/serverutil"

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		getSmbStatusRoute,
		setupSmbRoute,
		teardownSmbRoute,
	}
}
