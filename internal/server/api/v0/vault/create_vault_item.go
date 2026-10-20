package v0_vault

import (
	"errors"
	"net/http"

	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"

	"github.com/gin-gonic/gin"
)

// createVaultItem godoc
// @Summary Add an item to the caller's vault
// @Description Stores a new item, an entry or a folder, encrypted on the client under the caller's vault key: a non-empty ciphertext, a nonce of at most 64 bytes, both base64, and formatVersion, the client's plaintext layout version, at least 1. The body is at most 1 MiB. The caller must have set its vault up with PUT /vault/key first, or gets 409. Publishes vault_changed with the new item's id to the caller's sessions.
// @Tags vault
// @Accept json
// @Produce json
// @Param body body vaultutil.ItemFields true "The encrypted item"
// @Success 201 {object} vaultutil.Item
// @Failure 400 {object} serverutil.Response "an item of the wrong size or shape"
// @Failure 401 {object} serverutil.Response
// @Failure 409 {object} serverutil.Response "the caller has no vault yet"
// @Failure 413 {object} serverutil.Response "a body over 1 MiB"
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /vault/items [post]
func createVaultItem(c *gin.Context) *serverutil.Response {
	deps, userID, failed := callerContext(c)
	if failed != nil {
		return failed
	}
	var body vaultutil.ItemFields
	if failed := bindBody(c, vaultutil.MaxItemRequestBytes, &body, vaultutil.ErrInvalidItem); failed != nil {
		return failed
	}
	result, err := vaultutil.CreateItem(vaultutil.CreateItemParams{Ctx: c.Request.Context(), Database: deps.Database(), UserID: userID, Fields: body})
	if errors.Is(err, vaultutil.ErrKeyNotFound) {
		return serverutil.Conflict(err)
	}
	if err != nil {
		return vaultError(err)
	}
	vaultutil.PublishChanged(vaultutil.PublishChangedParams{EventBus: deps.EventBus(), UserID: userID, ItemID: result.Item.ID})
	return serverutil.NewResponse().WithStatusCode(http.StatusCreated).WithData(result.Item)
}

var createVaultItemRoute = serverutil.ApiRoute("POST", "/vault/items", createVaultItem)
