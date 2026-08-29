package v0_favorites

import (
	"github.com/autobutler-org/quark/pkg/util/serverutil"
)

type router struct{}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		toggleFavoriteRoute,
		isFavoriteRoute,
		listFavoritesRoute,
	}
}

type favoriteRequest struct {
	DeviceSerial string `json:"deviceSerial"`
	RelPath      string `json:"relPath"`
}

type favoriteResponse struct {
	IsFavorite bool `json:"isFavorite"`
}

type favoriteItemJSON struct {
	DeviceSerial string `json:"deviceSerial"`
	RelPath      string `json:"relPath"`
	CreatedAt    string `json:"createdAt"`
}
