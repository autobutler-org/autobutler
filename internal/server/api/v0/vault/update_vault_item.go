package v0_vault

import (
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"

	"github.com/gin-gonic/gin"
)

// updateVaultItem godoc
// @Summary Replace one of the caller's vault items
// @Description Replaces an item's ciphertext, nonce and formatVersion, with the body POST /vault/items takes, at most 1 MiB. Another account's item id, or one that names nothing, is a 404, for an admin too, and changes nothing. Publishes vault_changed with the item's id to the caller's sessions.
// @Tags vault
// @Accept json
// @Produce json
// @Param id path int true "Item id"
// @Param body body vaultutil.ItemFields true "The re-encrypted item"
// @Success 200 {object} vaultutil.Item
// @Failure 400 {object} serverutil.Response "an item of the wrong size or shape"
// @Failure 401 {object} serverutil.Response
// @Failure 404 {object} serverutil.Response "the caller has no item with that id"
// @Failure 413 {object} serverutil.Response "a body over 1 MiB"
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /vault/items/{id} [put]
func updateVaultItem(c *gin.Context) *serverutil.Response {
	deps, userID, failed := callerContext(c)
	if failed != nil {
		return failed
	}
	id, failed := itemID(c)
	if failed != nil {
		return failed
	}
	var body vaultutil.ItemFields
	if failed := bindBody(c, vaultutil.MaxItemRequestBytes, &body, vaultutil.ErrInvalidItem); failed != nil {
		return failed
	}
	result, err := vaultutil.UpdateItem(vaultutil.UpdateItemParams{Ctx: c.Request.Context(), Queries: deps.Database().Queries, UserID: userID, ID: id, Fields: body})
	if err != nil {
		return vaultError(err)
	}
	vaultutil.PublishChanged(vaultutil.PublishChangedParams{EventBus: deps.EventBus(), UserID: userID, ItemID: id})
	return serverutil.Ok().WithData(result.Item)
}

var updateVaultItemRoute = serverutil.ApiRoute("PUT", "/vault/items/:id", updateVaultItem)
