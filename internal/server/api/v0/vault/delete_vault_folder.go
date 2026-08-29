package v0_vault

import (
	"fmt"
	"strconv"

	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultcrypto"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"
	"github.com/gin-gonic/gin"
)

var deleteVaultFolderRoute = serverutil.ApiRoute(
	"DELETE", "/vault/folders/:id", func(c *gin.Context) *serverutil.Response {
		deps, key, errResp := requireUnlockedVault(c)
		if errResp != nil {
			return errResp
		}
		defer vaultcrypto.ZeroKey(key)

		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			return serverutil.BadRequest(fmt.Errorf("invalid id"))
		}

		result, err := vaultutil.DeleteFolder(c.Request.Context(), vaultutil.DeleteFolderParams{
			Queries: deps.VaultDB().Queries,
			ID:      id,
		})
		if err != nil {
			return serverutil.InternalServerError(err)
		}

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(gin.H{
			"deleted": result.Deleted,
		})
	},
)
