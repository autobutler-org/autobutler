package v0_vault

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultcrypto"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"
	"github.com/gin-gonic/gin"
)

var updateVaultEntryRoute = serverutil.ApiRoute(
	"PUT", "/vault/entries/:id", func(c *gin.Context) *serverutil.Response {
		deps, key, errResp := requireUnlockedVault(c)
		if errResp != nil {
			return errResp
		}
		defer vaultcrypto.ZeroKey(key)

		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			return serverutil.BadRequest(fmt.Errorf("invalid id"))
		}

		var req createEntryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.BadRequest(fmt.Errorf("invalid request: %w", err))
		}

		result, err := vaultutil.UpdateEntry(c.Request.Context(), vaultutil.UpdateEntryParams{
			Queries: deps.VaultDB().Queries,
			Key:     key,
			ID:      id,
			Fields:  req.fields(),
		})
		if errors.Is(err, vaultutil.ErrEntryNotFound) {
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
