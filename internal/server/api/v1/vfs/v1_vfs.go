package v1_vfs

import "github.com/autobutler-org/quark/pkg/util/serverutil"

type router struct{}

// NewRouter returns the VFS API v1 router.
func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		listNamespacesRoute,
		// vfsQueryMetaRoute is intentionally NOT a separate route:
		// registering GET /vfs/:ns/_meta/query alongside GET /vfs/:ns/*path
		// would cause a Gin wildcard conflict panic at startup.
		// The query dispatch is handled inside vfsReadRoute instead.
		vfsReadRoute,
		vfsWriteRoute,
		vfsDeleteRoute,
		vfsMkdirRoute,
	}
}
