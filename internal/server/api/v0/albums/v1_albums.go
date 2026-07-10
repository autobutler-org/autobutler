package v0_albums

import "github.com/autobutler-org/autobutler/pkg/util/serverutil"

// Router for /api/v1/albums endpoints

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		listAlbumsRoute,
		getAlbumRoute,
		createAlbumRoute,
		renameAlbumRoute,
		moveAlbumRoute,
		deleteAlbumRoute,
		listAlbumItemsRoute,
		addPhotoToAlbumRoute,
		removePhotoFromAlbumRoute,
	}
}
