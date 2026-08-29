package v0_vault

import (
	"fmt"

	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultcrypto"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"
	"github.com/gin-gonic/gin"
)

var createVaultEntryRoute = serverutil.ApiRoute(
	"POST", "/vault/entries", func(c *gin.Context) *serverutil.Response {
		deps, key, errResp := requireUnlockedVault(c)
		if errResp != nil {
			return errResp
		}
		defer vaultcrypto.ZeroKey(key)

		var req createEntryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.BadRequest(fmt.Errorf("invalid request: %w", err))
		}

		result, err := vaultutil.CreateEntry(c.Request.Context(), vaultutil.CreateEntryParams{
			Queries: deps.VaultDB().Queries,
			Key:     key,
			Fields:  req.fields(),
		})
		if err != nil {
			return serverutil.InternalServerError(err)
		}

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(result.Entry)
	},
)
