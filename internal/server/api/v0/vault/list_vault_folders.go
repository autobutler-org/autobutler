package v0_vault

import (
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"
	"github.com/gin-gonic/gin"
)

var listVaultFoldersRoute = serverutil.ApiRoute(
	"GET", "/vault/folders", func(c *gin.Context) *serverutil.Response {
		deps, errResp := getDeps(c)
		if errResp != nil {
			return errResp
		}

		result, err := vaultutil.ListFolders(c.Request.Context(), vaultutil.ListFoldersParams{
			Queries: deps.VaultDB().Queries,
		})
		if err != nil {
			return serverutil.InternalServerError(err)
		}

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(gin.H{
			"folders": result.Folders,
		})
	},
)
