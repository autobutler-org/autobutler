package v0_vault

import (
	"net/http"

	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"

	"github.com/gin-gonic/gin"
)

// deleteVault godoc
// @Summary Delete the caller's vault
// @Description Deletes every item of the caller's vault and its vault key in one transaction, so an account locked out of its vault, having lost both its password and its recovery phrase, can start over with PUT /vault/key. An account with no vault gets 204 too. Publishes vault_changed to the caller's sessions.
// @Tags vault
// @Success 204
// @Failure 401 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /vault [delete]
func deleteVault(c *gin.Context) *serverutil.Response {
	deps, userID, failed := callerContext(c)
	if failed != nil {
		return failed
	}
	if _, err := vaultutil.DeleteVault(vaultutil.DeleteVaultParams{Ctx: c.Request.Context(), Database: deps.Database(), UserID: userID}); err != nil {
		return vaultError(err)
	}
	vaultutil.PublishChanged(vaultutil.PublishChangedParams{EventBus: deps.EventBus(), UserID: userID})
	return serverutil.NewResponse().WithStatusCode(http.StatusNoContent)
}

var deleteVaultRoute = serverutil.ApiRoute("DELETE", "/vault", deleteVault)
