package v0_favorites

import (
	"errors"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/favoritesutil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/sqlutil"
	"github.com/gin-gonic/gin"
)

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

// toggleFavorite godoc
// @Summary Toggle a photo favorite
// @Description Adds the photo to favorites if not already favorited; removes it otherwise.
// @Tags favorites
// @Accept json
// @Produce json
// @Param body body favoriteRequest true "Photo reference"
// @Success 200 {object} favoriteResponse
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /photos/favorite [post]
func toggleFavorite(c *gin.Context) *serverutil.Response {
	var req favoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(errors.New("invalid request body"))
	}
	if req.RelPath == "" {
		return serverutil.BadRequest(errors.New("relPath is required"))
	}

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	isFav, err := favoritesutil.ToggleFavorite(
		c.Request.Context(),
		deps.Database().Queries,
		req.DeviceSerial,
		req.RelPath,
	)
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(favoriteResponse{
		IsFavorite: isFav,
	})
}

// isFavorite godoc
// @Summary Check if a photo is favorited
// @Description Returns whether the specified photo is in the user's favorites.
// @Tags favorites
// @Produce json
// @Param serial query string false "Device serial"
// @Param relPath query string true "Relative path to the photo"
// @Success 200 {object} favoriteResponse
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /photos/favorite [get]
func isFavorite(c *gin.Context) *serverutil.Response {
	relPath := c.Query("relPath")
	if relPath == "" {
		return serverutil.BadRequest(errors.New("relPath is required"))
	}
	serial := c.Query("serial")

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	fav, err := deps.Database().Queries.IsFavorite(c.Request.Context(), db.IsFavoriteParams{
		DeviceSerial: serial,
		RelPath:      relPath,
	})
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(favoriteResponse{
		IsFavorite: fav,
	})
}

// listFavorites godoc
// @Summary List all favorited photos
// @Description Returns all photos the user has favorited, newest first.
// @Tags favorites
// @Produce json
// @Success 200 {array} favoriteItemJSON
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /photos/favorites [get]
func listFavorites(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	items, err := deps.Database().Queries.ListFavorites(c.Request.Context())
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	result := make([]favoriteItemJSON, 0, len(items))
	for _, item := range items {
		result = append(result, favoriteItemJSON{
			DeviceSerial: item.DeviceSerial,
			RelPath:      item.RelPath,
			CreatedAt:    sqlutil.FormatTime(item.CreatedAt),
		})
	}

	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(result)
}

var toggleFavoriteRoute = serverutil.ApiRoute("POST", "/photos/favorite", toggleFavorite)
var isFavoriteRoute = serverutil.ApiRoute("GET", "/photos/favorite", isFavorite)
var listFavoritesRoute = serverutil.ApiRoute("GET", "/photos/favorites", listFavorites)
