package v1_albums

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/sqlutil"

	"github.com/gin-gonic/gin"
)

type renameAlbumRequest struct {
	Name string `json:"name" binding:"required"`
}

// renameAlbum godoc
// @Summary Rename a photo album
// @Description Updates the name of an existing album.
// @Tags albums
// @Accept json
// @Produce json
// @Param id path int true "Album ID"
// @Param body body renameAlbumRequest true "New album name"
// @Success 200 {object} AlbumJSON
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /albums/{id}/rename [patch]
func renameAlbum(c *gin.Context) *serverutil.Response {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return serverutil.BadRequest(errors.New("invalid album id"))
	}

	var req renameAlbumRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(errors.New("name is required"))
	}

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	album, err := deps.Database().Queries.RenameAlbum(context.Background(), db.RenameAlbumParams{
		Name: req.Name,
		ID:   id,
	})
	if err != nil {
		return serverutil.NotFound(err)
	}

	var parentID *int64
	if album.ParentID.Valid {
		parentID = &album.ParentID.Int64
	}

	count, _ := deps.Database().Queries.CountAlbumItems(context.Background(), album.ID)

	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(AlbumJSON{
		ID:        album.ID,
		Name:      album.Name,
		ParentID:  parentID,
		CreatedAt: sqlutil.FormatTime(album.CreatedAt),
		UpdatedAt: sqlutil.FormatTime(album.UpdatedAt),
		ItemCount: count,
	})
}

type moveAlbumRequest struct {
	ParentID *int64 `json:"parentId"`
}

// moveAlbum godoc
// @Summary Move a photo album to a new parent
// @Description Changes the parent of an album. Pass null parentId to move to root.
// @Tags albums
// @Accept json
// @Produce json
// @Param id path int true "Album ID"
// @Param body body moveAlbumRequest true "New parent ID (null for root)"
// @Success 200 {object} AlbumJSON
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /albums/{id}/move [patch]
func moveAlbum(c *gin.Context) *serverutil.Response {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return serverutil.BadRequest(errors.New("invalid album id"))
	}

	var req moveAlbumRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(errors.New("invalid request body"))
	}

	if req.ParentID != nil && *req.ParentID == id {
		return serverutil.BadRequest(errors.New("album cannot be its own parent"))
	}

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	var parentID sql.NullInt64
	if req.ParentID != nil {
		if _, err := deps.Database().Queries.GetAlbum(context.Background(), *req.ParentID); err != nil {
			return serverutil.BadRequest(errors.New("parent album not found"))
		}
		parentID = sql.NullInt64{Int64: *req.ParentID, Valid: true}
	}

	album, err := deps.Database().Queries.MoveAlbum(context.Background(), db.MoveAlbumParams{
		ParentID: parentID,
		ID:       id,
	})
	if err != nil {
		return serverutil.NotFound(err)
	}

	var respParentID *int64
	if album.ParentID.Valid {
		respParentID = &album.ParentID.Int64
	}

	count, _ := deps.Database().Queries.CountAlbumItems(context.Background(), album.ID)

	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(AlbumJSON{
		ID:        album.ID,
		Name:      album.Name,
		ParentID:  respParentID,
		CreatedAt: sqlutil.FormatTime(album.CreatedAt),
		UpdatedAt: sqlutil.FormatTime(album.UpdatedAt),
		ItemCount: count,
	})
}

var renameAlbumRoute = serverutil.ApiRoute(
	"PATCH", "/albums/:id/rename", renameAlbum,
)

var moveAlbumRoute = serverutil.ApiRoute(
	"PATCH", "/albums/:id/move", moveAlbum,
)
