package v0_smb

import (
	"github.com/autobutler-org/quark/pkg/util/serverutil"
)

type router struct{}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		getSmbStatusRoute,
		setupSmbRoute,
		teardownSmbRoute,
	}
}
