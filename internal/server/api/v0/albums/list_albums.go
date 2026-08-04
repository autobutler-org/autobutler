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
// Uses an internal linked-node structure so child-of-child relationships are
// captured correctly regardless of depth, then serialises back to []AlbumJSON.
func buildTree(albums []AlbumJSON) []AlbumJSON {
	type treeNode struct {
		alb      AlbumJSON
		children []*treeNode
	}

	byID := make(map[int64]*treeNode, len(albums))
	nodes := make([]*treeNode, len(albums))
	for i, a := range albums {
		n := &treeNode{alb: a}
		nodes[i] = n
		byID[a.ID] = n
	}

	var roots []*treeNode
	for _, n := range nodes {
		if n.alb.ParentID == nil {
			roots = append(roots, n)
		} else if parent, ok := byID[*n.alb.ParentID]; ok {
			parent.children = append(parent.children, n)
		}
		// nodes whose parent isn't in the list are silently dropped
	}

	var toJSON func(*treeNode) AlbumJSON
	toJSON = func(n *treeNode) AlbumJSON {
		a := n.alb
		if len(n.children) > 0 {
			a.Children = make([]AlbumJSON, 0, len(n.children))
			for _, child := range n.children {
				a.Children = append(a.Children, toJSON(child))
			}
		}
		return a
	}

	result := make([]AlbumJSON, 0, len(roots))
	for _, r := range roots {
		result = append(result, toJSON(r))
	}
	return result
}

var listAlbumsRoute = serverutil.ApiRoute(
	"GET", "/albums", listAlbums,
)
