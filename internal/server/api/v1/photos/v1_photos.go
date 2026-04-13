package v1_photos

import "github.com/autobutler-org/autobutler/pkg/util/serverutil"

// Router for /api/v1/photos endpoints
// Registers the /photos route

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		listPhotosRoute,
		getPhotoMetadataRoute,
	}
}
