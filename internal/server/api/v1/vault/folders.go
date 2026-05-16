package v1_vault

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

type folderJSON struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	ParentID  *int64 `json:"parentId"`
	SortOrder int64  `json:"sortOrder"`
	CreatedAt string `json:"createdAt"`
}

type createFolderRequest struct {
	Name      string `json:"name" binding:"required"`
	ParentID  *int64 `json:"parentId"`
	SortOrder int64  `json:"sortOrder"`
}

var listFoldersRoute = serverutil.ApiRoute(
	"GET", "/vault/folders", func(c *gin.Context) *serverutil.Response {
		deps, errResp := getDeps(c)
		if errResp != nil {
			return errResp
		}

		rows, err := deps.Database().Queries.ListVaultFolders(c.Request.Context())
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("list folders: %w", err))
		}

		folders := make([]folderJSON, 0, len(rows))
		for _, r := range rows {
			folders = append(folders, folderJSON{
				ID:        r.ID,
				Name:      r.Name,
				ParentID:  fromNullInt64(r.ParentID),
				SortOrder: r.SortOrder,
				CreatedAt: r.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			})
		}

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(gin.H{
			"folders": folders,
		})
	},
)

var createFolderRoute = serverutil.ApiRoute(
	"POST", "/vault/folders", func(c *gin.Context) *serverutil.Response {
		deps, errResp := getDeps(c)
		if errResp != nil {
			return errResp
		}

		var req createFolderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.BadRequest(fmt.Errorf("invalid request: %w", err))
		}

		folder, err := deps.Database().Queries.CreateVaultFolder(c.Request.Context(), db.CreateVaultFolderParams{
			Name:      req.Name,
			ParentID:  nullableInt64(req.ParentID),
			SortOrder: req.SortOrder,
		})
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("create folder: %w", err))
		}

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(folderJSON{
			ID:        folder.ID,
			Name:      folder.Name,
			ParentID:  fromNullInt64(folder.ParentID),
			SortOrder: folder.SortOrder,
			CreatedAt: folder.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	},
)

var updateFolderRoute = serverutil.ApiRoute(
	"PUT", "/vault/folders/:id", func(c *gin.Context) *serverutil.Response {
		deps, errResp := getDeps(c)
		if errResp != nil {
			return errResp
		}

		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			return serverutil.BadRequest(fmt.Errorf("invalid id"))
		}

		ctx := c.Request.Context()

		if _, err := deps.Database().Queries.GetVaultFolder(ctx, id); errors.Is(err, sql.ErrNoRows) {
			return serverutil.NotFound(fmt.Errorf("folder not found"))
		} else if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("get folder: %w", err))
		}

		var req createFolderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.BadRequest(fmt.Errorf("invalid request: %w", err))
		}

		if err := deps.Database().Queries.UpdateVaultFolder(ctx, db.UpdateVaultFolderParams{
			Name:      req.Name,
			ParentID:  nullableInt64(req.ParentID),
			SortOrder: req.SortOrder,
			ID:        id,
		}); err != nil {
			return serverutil.InternalServerError(fmt.Errorf("update folder: %w", err))
		}

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(gin.H{
			"id": id,
		})
	},
)

var deleteFolderRoute = serverutil.ApiRoute(
	"DELETE", "/vault/folders/:id", func(c *gin.Context) *serverutil.Response {
		deps, errResp := getDeps(c)
		if errResp != nil {
			return errResp
		}

		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			return serverutil.BadRequest(fmt.Errorf("invalid id"))
		}

		if err := deps.Database().Queries.DeleteVaultFolder(c.Request.Context(), id); err != nil {
			return serverutil.InternalServerError(fmt.Errorf("delete folder: %w", err))
		}

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(gin.H{
			"deleted": true,
		})
	},
)
