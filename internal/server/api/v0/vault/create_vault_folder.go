package v0_vault

import (
	"fmt"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

var createVaultFolderRoute = serverutil.ApiRoute(
	"POST", "/vault/folders", func(c *gin.Context) *serverutil.Response {
		deps, errResp := getDeps(c)
		if errResp != nil {
			return errResp
		}

		var req createFolderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.BadRequest(fmt.Errorf("invalid request: %w", err))
		}

		folder, err := deps.VaultDB().Queries.CreateVaultFolder(c.Request.Context(), db.CreateVaultFolderParams{
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
