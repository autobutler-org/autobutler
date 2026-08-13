package v0_files

import "github.com/autobutler-org/autobutler/pkg/util/serverutil"

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		deleteFilesRoute,
		downloadArchiveFileRoute,
		downloadFileRoute,
		extractFileRoute,
		listArchiveRoute,
		listFilesByTypeRoute,
		listFilesRoute,
		listRecentFilesRoute,
		moveFileRoute,
		newFolderRoute,
		searchFilesRoute,
		searchContentRoute,
		statFileRoute,
		uploadFilesRoute,
		uploadFilesNestedRoute,
	}
}
