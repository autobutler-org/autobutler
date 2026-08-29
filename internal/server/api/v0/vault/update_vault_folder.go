package v0_vault

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"
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

		var req createFolderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.BadRequest(fmt.Errorf("invalid request: %w", err))
		}

		result, err := vaultutil.UpdateFolder(c.Request.Context(), vaultutil.UpdateFolderParams{
			Queries: deps.VaultDB().Queries,
			ID:      id,
			Fields:  req.fields(),
		})
		if errors.Is(err, vaultutil.ErrFolderNotFound) {
			return serverutil.NotFound(err)
		}
		if err != nil {
			return serverutil.InternalServerError(err)
		}

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(gin.H{
			"id": result.ID,
		})
	},
)
