package v1_vfs

import "github.com/autobutler-org/autobutler/pkg/util/serverutil"

type router struct{}

// NewRouter returns the VFS API v1 router.
func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		listNamespacesRoute,
		vfsQueryMetaRoute, // static route — must be registered before the wildcard
		vfsReadRoute,
		vfsWriteRoute,
		vfsDeleteRoute,
		vfsMkdirRoute,
	}
}
