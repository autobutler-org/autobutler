package v1_files

import "autobutler/pkg/util/serverutil"

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		deleteFilesRoute,
		newFolderRoute,
		moveFileRoute,
		uploadNestedFilesRoutes,
		uploadRootFilesRoute,
		cirrusRoute,
		cirrusNestedRoute,
	}
}
