package v0_vault

import (
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"

	"github.com/gin-gonic/gin"
)

// listVaultItems godoc
// @Summary List the caller's vault items
// @Description Returns every item of the caller's vault, entries and folders alike, oldest first, as stored: ciphertext and nonce in base64 and the client's formatVersion. What an item is lives inside its ciphertext. An account with no vault gets an empty list.
// @Tags vault
// @Produce json
// @Success 200 {array} vaultutil.Item
// @Failure 401 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /vault/items [get]
func listVaultItems(c *gin.Context) *serverutil.Response {
	deps, userID, failed := callerContext(c)
	if failed != nil {
		return failed
	}
	result, err := vaultutil.ListItems(vaultutil.ListItemsParams{Ctx: c.Request.Context(), Queries: deps.Database().Queries, UserID: userID})
	if err != nil {
		return vaultError(err)
	}
	return serverutil.Ok().WithData(result.Items)
}

var listVaultItemsRoute = serverutil.ApiRoute("GET", "/vault/items", listVaultItems)
