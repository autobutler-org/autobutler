package v0_vault

import (
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"
	"github.com/gin-gonic/gin"
)

var listVaultEntriesRoute = serverutil.ApiRoute(
	"GET", "/vault/entries", func(c *gin.Context) *serverutil.Response {
		deps, errResp := getDeps(c)
		if errResp != nil {
			return errResp
		}

		result, err := vaultutil.ListEntries(c.Request.Context(), vaultutil.ListEntriesParams{
			Queries: deps.VaultDB().Queries,
		})
		if err != nil {
			return serverutil.InternalServerError(err)
		}

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(gin.H{
			"entries": result.Entries,
		})
	},
)
