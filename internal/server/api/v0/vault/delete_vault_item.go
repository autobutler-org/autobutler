package v0_vault

import (
	"net/http"

	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"

	"github.com/gin-gonic/gin"
)

// deleteVaultItem godoc
// @Summary Delete one of the caller's vault items
// @Description Deletes one item of the caller's vault. Another account's item id, or one that names nothing, is a 404, for an admin too, and deletes nothing. Publishes vault_changed with the item's id to the caller's sessions.
// @Tags vault
// @Param id path int true "Item id"
// @Success 204
// @Failure 401 {object} serverutil.Response
// @Failure 404 {object} serverutil.Response "the caller has no item with that id"
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /vault/items/{id} [delete]
func deleteVaultItem(c *gin.Context) *serverutil.Response {
	deps, userID, failed := callerContext(c)
	if failed != nil {
		return failed
	}
	id, failed := itemID(c)
	if failed != nil {
		return failed
	}
	if _, err := vaultutil.DeleteItem(vaultutil.DeleteItemParams{Ctx: c.Request.Context(), Queries: deps.Database().Queries, UserID: userID, ID: id}); err != nil {
		return vaultError(err)
	}
	vaultutil.PublishChanged(vaultutil.PublishChangedParams{EventBus: deps.EventBus(), UserID: userID, ItemID: id})
	return serverutil.NewResponse().WithStatusCode(http.StatusNoContent)
}

var deleteVaultItemRoute = serverutil.ApiRoute("DELETE", "/vault/items/:id", deleteVaultItem)
