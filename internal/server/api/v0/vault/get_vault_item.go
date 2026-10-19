package v0_vault

import (
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"

	"github.com/gin-gonic/gin"
)

// getVaultItem godoc
// @Summary Get one of the caller's vault items
// @Description Returns one item of the caller's vault as stored. Another account's item id, or one that names nothing, is a 404, for an admin too.
// @Tags vault
// @Produce json
// @Param id path int true "Item id"
// @Success 200 {object} vaultutil.Item
// @Failure 401 {object} serverutil.Response
// @Failure 404 {object} serverutil.Response "the caller has no item with that id"
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /vault/items/{id} [get]
func getVaultItem(c *gin.Context) *serverutil.Response {
	deps, userID, failed := callerContext(c)
	if failed != nil {
		return failed
	}
	id, failed := itemID(c)
	if failed != nil {
		return failed
	}
	result, err := vaultutil.GetItem(vaultutil.GetItemParams{Ctx: c.Request.Context(), Queries: deps.Database().Queries, UserID: userID, ID: id})
	if err != nil {
		return vaultError(err)
	}
	return serverutil.Ok().WithData(result.Item)
}

var getVaultItemRoute = serverutil.ApiRoute("GET", "/vault/items/:id", getVaultItem)
