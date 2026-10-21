package v0_vault

import (
	"errors"
	"net/http"

	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"

	"github.com/gin-gonic/gin"
)

// deleteVault godoc
// @Summary Delete the caller's vault
// @Description Re-confirms the caller with password, which carries the auth key of the login password, exactly as PUT /vault/key takes it; a wrong or missing key is a 403, a raw password is a 426, and nothing is deleted, so a session alone cannot wipe a vault (#3615). The body is at most 1 KiB. Then deletes every item of the caller's vault and its vault key in one transaction, so an account locked out of its vault, having lost both its password and its recovery phrase, can start over with PUT /vault/key. An account with no vault gets 204 too. Publishes vault_changed to the caller's sessions. Shares the sign-in rate limit.
// @Tags vault
// @Accept json
// @Param body body deleteVaultBody true "The caller's auth key"
// @Success 204
// @Failure 400 {object} serverutil.Response "a body that is not JSON"
// @Failure 401 {object} serverutil.Response
// @Failure 403 {object} serverutil.Response "password is not the caller's auth key"
// @Failure 413 {object} serverutil.Response "a body over 1 KiB"
// @Failure 426 {object} serverutil.Response "password is a raw password: the app is too old"
// @Failure 429 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /vault [delete]
func deleteVault(c *gin.Context) *serverutil.Response {
	deps, userID, failed := callerContext(c)
	if failed != nil {
		return failed
	}
	var body deleteVaultBody
	if failed := bindBody(c, maxDeleteVaultRequestBytes, &body, errPasswordRequired); failed != nil {
		return failed
	}
	// Nothing goes before this: the wipe cannot be undone.
	if failed := reconfirm(c, deps, body.Password); failed != nil {
		return failed
	}
	if _, err := vaultutil.DeleteVault(vaultutil.DeleteVaultParams{Ctx: c.Request.Context(), Database: deps.Database(), UserID: userID}); err != nil {
		return vaultError(err)
	}
	vaultutil.PublishChanged(vaultutil.PublishChangedParams{EventBus: deps.EventBus(), UserID: userID})
	return serverutil.NewResponse().WithStatusCode(http.StatusNoContent)
}

// maxDeleteVaultRequestBytes bounds DELETE /vault's body, which holds only
// an auth key.
const maxDeleteVaultRequestBytes = 1 << 10

// errPasswordRequired is the 400 a DELETE /vault body that is not JSON gets.
var errPasswordRequired = errors.New("password is required")

var deleteVaultRoute = serverutil.ApiRoute("DELETE", "/vault", deleteVault)
