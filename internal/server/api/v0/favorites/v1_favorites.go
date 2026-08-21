package v0_favorites

import "github.com/autobutler-org/quark/pkg/util/serverutil"

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		toggleFavoriteRoute,
		isFavoriteRoute,
		listFavoritesRoute,
	}
}
