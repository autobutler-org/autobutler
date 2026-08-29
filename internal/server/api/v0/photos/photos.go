package v0_photos

import "github.com/autobutler-org/quark/pkg/util/serverutil"

// Router for /api/v0/photos endpoints
// Registers the /photos route

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		listPhotosRoute,
		getPhotoMetadataRoute,
		rotatePhotoRoute,
		copyPhotoRoute,
		listDuplicatesRoute,
	}
}
