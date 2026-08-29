package v0_vault

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

var updateVaultFolderRoute = serverutil.ApiRoute(
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

		if _, err := deps.VaultDB().Queries.GetVaultFolder(ctx, id); errors.Is(err, sql.ErrNoRows) {
			return serverutil.NotFound(fmt.Errorf("folder not found"))
		} else if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("get folder: %w", err))
		}

		var req createFolderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.BadRequest(fmt.Errorf("invalid request: %w", err))
		}

		if err := deps.VaultDB().Queries.UpdateVaultFolder(ctx, db.UpdateVaultFolderParams{
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
