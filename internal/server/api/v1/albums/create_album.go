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

type createAlbumRequest struct {
	Name     string `json:"name" binding:"required"`
	ParentID *int64 `json:"parentId"`
}

// createAlbum godoc
// @Summary Create a photo album
// @Description Creates a new photo album, optionally nested under a parent.
// @Tags albums
// @Accept json
// @Produce json
// @Param body body createAlbumRequest true "Album name and optional parent ID"
// @Success 201 {object} AlbumJSON
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /albums [post]
func createAlbum(c *gin.Context) *serverutil.Response {
	var req createAlbumRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(errors.New("name is required"))
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

	album, err := deps.Database().Queries.CreateAlbum(context.Background(), db.CreateAlbumParams{
		Name:     req.Name,
		ParentID: parentID,
	})
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	var respParentID *int64
	if album.ParentID.Valid {
		respParentID = &album.ParentID.Int64
	}

	return serverutil.Ok().WithStatusCode(201).WithContentType(serverutil.ContentTypeJSON).WithData(AlbumJSON{
		ID:        album.ID,
		Name:      album.Name,
		ParentID:  respParentID,
		CreatedAt: sqlutil.FormatTime(album.CreatedAt),
		UpdatedAt: sqlutil.FormatTime(album.UpdatedAt),
		ItemCount: 0,
	})
}

// createChildAlbum godoc
// @Summary Create a child album under a parent
// @Description Shorthand for creating an album with a specific parent ID.
// @Tags albums
// @Accept json
// @Produce json
// @Param id path int true "Parent album ID"
// @Param body body createAlbumRequest true "Album name"
// @Success 201 {object} AlbumJSON
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /albums/{id}/children [post]
func createChildAlbum(c *gin.Context) *serverutil.Response {
	parentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return serverutil.BadRequest(errors.New("invalid album id"))
	}
	c.Set("_override_parent_id", parentID)
	return createAlbum(c)
}

var createAlbumRoute = serverutil.ApiRoute(
	"POST", "/albums", createAlbum,
)
