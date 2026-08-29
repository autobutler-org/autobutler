package v0_vault

import (
	"fmt"

	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"
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

		result, err := vaultutil.CreateFolder(c.Request.Context(), vaultutil.CreateFolderParams{
			Queries: deps.VaultDB().Queries,
			Fields:  req.fields(),
		})
		if err != nil {
			return serverutil.InternalServerError(err)
		}

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(result.Folder)
	},
)
