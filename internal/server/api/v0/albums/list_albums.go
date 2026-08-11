package v0_albums

import (
	"context"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/sqlutil"

	"github.com/gin-gonic/gin"
)

// listAlbums godoc
// @Summary List all photo albums
// @Description Returns all photo albums as a flat list. Use ?tree=true to get a nested tree.
// @Tags albums
// @Produce json
// @Param tree query bool false "Return as nested tree (default false)"
// @Success 200 {array} AlbumJSON
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /albums [get]
func listAlbums(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	albums, err := deps.Database().Queries.ListAlbums(context.Background())
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	result := make([]AlbumJSON, 0, len(albums))
	for _, a := range albums {
		count, _ := deps.Database().Queries.CountAlbumItems(context.Background(), a.ID)
		var parentID *int64
		if a.ParentID.Valid {
			parentID = &a.ParentID.Int64
		}
		result = append(result, AlbumJSON{
			ID:        a.ID,
			Name:      a.Name,
			ParentID:  parentID,
			SmartType: sqlutil.NullStringPtr(a.SmartType),
			CreatedAt: sqlutil.FormatTime(a.CreatedAt),
			UpdatedAt: sqlutil.FormatTime(a.UpdatedAt),
			ItemCount: count,
		})
	}

	wantTree := c.Query("tree") == "true"
	if wantTree {
		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(buildTree(result))
	}
	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(result)
}

// buildTree converts a flat album list into a nested tree.
// Uses a child-index map to avoid value-copy aliasing issues when building
// multi-level hierarchies.
func buildTree(albums []AlbumJSON) []AlbumJSON {
	// Map album ID → its children IDs.
	childIDs := make(map[int64][]int64, len(albums))
	byID := make(map[int64]AlbumJSON, len(albums))
	for _, a := range albums {
		byID[a.ID] = a
	}
	for _, a := range albums {
		if a.ParentID != nil {
			childIDs[*a.ParentID] = append(childIDs[*a.ParentID], a.ID)
		}
	}

	// build recursively populates Children before returning a copy.
	var build func(id int64) AlbumJSON
	build = func(id int64) AlbumJSON {
		a := byID[id]
		for _, cid := range childIDs[id] {
			a.Children = append(a.Children, build(cid))
		}
		return a
	}

	roots := []AlbumJSON{}
	for _, a := range albums {
		if a.ParentID == nil {
			roots = append(roots, build(a.ID))
		}
	}
	return roots
}

var listAlbumsRoute = serverutil.ApiRoute(
	"GET", "/albums", listAlbums,
)
