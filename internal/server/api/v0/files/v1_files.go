package v0_files

import "github.com/autobutler-org/quark/pkg/util/serverutil"

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	canonical := []*serverutil.Route{
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
		createUploadSessionRoute,
		uploadSessionChunkRoute,
		getUploadSessionRoute,
		deleteUploadSessionRoute,
	}

	// TODO(pre-v1.0.0, #1601): delete these two lines and return canonical.
	// Pre-rename clients still call /cirrus/*. Deprecated shim — see
	// legacy_cirrus_alias.go (#1601).
	aliases := legacyAliasRoutes(canonical)

	routes := make([]*serverutil.Route, 0, len(canonical)+len(aliases))
	routes = append(routes, canonical...)
	routes = append(routes, aliases...)
	return routes
}
