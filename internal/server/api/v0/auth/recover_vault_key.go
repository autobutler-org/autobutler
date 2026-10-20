package v0_auth

import (
	"errors"
	"net/http"

	"github.com/autobutler-org/quark/pkg/util/authutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"

	"github.com/gin-gonic/gin"
)

// recoverVaultKey godoc
// @Summary Fetch the vault key for account recovery
// @Description The vault's half of the first step of recovering an account (#3459), beside POST /auth/recover/keys for chat. Checks the recovery key exactly as /auth/recover does, changes nothing, and returns the account's wrapped vault key so the client can open it with the phrase wrap key and send it back re-wrapped under the new password as vaultKey in /auth/recover. Needs no session and shares the sign-in rate limit. An unknown username or a wrong key reads as a wrong phrase. A legacy account, one GET /auth/salt calls legacyRecovery, may send its raw recoveryPhrase in place of recoveryKey, as /auth/recover/keys takes it.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body recoverChatKeysBody true "The account and its recovery key, or a legacy account's phrase"
// @Success 200 {object} vaultutil.Key
// @Failure 400 {object} serverutil.Response "a wrong key or phrase, an unknown username, or a missing or malformed key"
// @Failure 403 {object} accountRefusal "status is pending or disabled"
// @Failure 404 {object} serverutil.Response "the account has no vault"
// @Failure 429 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Router /auth/recover/vault-key [post]
func recoverVaultKey(c *gin.Context) *serverutil.Response {
	deps, ok := getQueries(c)
	if !ok || (*deps).Database() == nil {
		return serverutil.InternalServerError(nil)
	}
	var req recoverChatKeysBody
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(err)
	}
	queries := (*deps).Database().Queries
	checked, err := authutil.CheckRecovery(c.Request.Context(), queries, authutil.CheckRecoveryParams{
		Username:       req.Username,
		RecoveryPhrase: req.RecoveryPhrase,
		RecoveryKey:    req.RecoveryKey,
	})
	if refusal := accountRefusalResponse(err); refusal != nil {
		return refusal
	}
	if err != nil {
		return serverutil.BadRequest(err)
	}
	result, err := vaultutil.GetKey(vaultutil.GetKeyParams{Ctx: c.Request.Context(), Queries: queries, UserID: checked.UserID})
	if errors.Is(err, vaultutil.ErrKeyNotFound) {
		return serverutil.NewResponse().WithStatusCode(http.StatusNotFound).WithError(err)
	}
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	return serverutil.Ok().WithData(result.Key)
}

var recoverVaultKeyRoute = serverutil.ApiRoute("POST", "/auth/recover/vault-key", recoverVaultKey)
