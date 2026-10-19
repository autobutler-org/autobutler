package v0_auth

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/autobutler-org/quark/pkg/util/authutil"
	"github.com/autobutler-org/quark/pkg/util/chatutil"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

// changePassword godoc
// @Summary Change the caller's password
// @Description Re-confirms the caller with password, which carries the auth key of the password the account has now, exactly as PUT /auth/recovery-key takes it; a wrong or missing key is a 403, a raw password is a 426, and nothing is written. Then replaces the account's auth key with newAuthKey, the standard base64 of the 32-byte key the client derived from the new password and the salt GET /auth/salt returned (#3022). chatKeys, when sent, replaces the account's chat identity in the same transaction, re-wrapped under the new password, exactly as /auth/recover stores it, and vaultKey likewise replaces the wraps of the account's vault key (storing it as the first when the account has none) and publishes vault_changed to the account's sessions; nothing changes if any part fails. Every other session of the account is signed out in the same transaction; this one stays signed in. Shares the sign-in rate limit, and the body is at most 8 KiB.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body changePasswordBody true "The caller's auth key, the new password's auth key and optionally the re-wrapped chat keys and vault key"
// @Success 204
// @Failure 400 {object} serverutil.Response "a malformed newAuthKey, chatKeys or vaultKey"
// @Failure 401 {object} serverutil.Response "no session, or the session's account no longer exists"
// @Failure 403 {object} serverutil.Response "the auth key is wrong or missing"
// @Failure 426 {object} serverutil.Response "password is a raw password: the app is too old"
// @Failure 429 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /auth/password [put]
func changePassword(c *gin.Context) *serverutil.Response {
	deps, ok := getQueries(c)
	if !ok || (*deps).Database() == nil {
		return serverutil.InternalServerError(nil)
	}
	userID, ok := ctxutil.Get[int64](c, "userID")
	if !ok || userID == 0 {
		return serverutil.Unauthorized(fmt.Errorf("not authenticated"))
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, chatutil.MaxKeysRequestBytes)
	var req changePasswordBody
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(err)
	}
	// Only the shape of the keys is checked here. ChangePassword checks the
	// caller's auth key before it or this step writes anything.
	afterChange, announce, err := storeRewrappedKeys(c.Request.Context(), *deps, req.ChatKeys, req.VaultKey)
	if err != nil {
		return serverutil.BadRequest(err)
	}

	// Absent under HTTP Basic auth, which has no session to keep.
	sessionID, _ := ctxutil.Get[string](c, "sessionID")
	_, err = authutil.ChangePassword(c.Request.Context(), (*deps).Database(), authutil.ChangePasswordParams{
		UserID:         userID,
		CurrentAuthKey: req.Password,
		NewAuthKey:     req.NewAuthKey,
		AfterChange:    afterChange,
		KeepSessionID:  sessionID,
	})
	switch {
	case errors.Is(err, authutil.ErrIncorrectPassword):
		return serverutil.NewResponse().WithStatusCode(http.StatusForbidden).WithError(err)
	case errors.Is(err, authutil.ErrInvalidAuthKey):
		return serverutil.BadRequest(err)
	case errors.Is(err, authutil.ErrUserNotFound):
		return serverutil.Unauthorized(fmt.Errorf("not authenticated"))
	case err != nil:
		// ErrAppTooOld is a 426 HttpError, whatever status this names.
		return serverutil.InternalServerError(err)
	}

	announce()
	return serverutil.NewResponse().WithStatusCode(http.StatusNoContent)
}

var changePasswordRoute = serverutil.ApiRoute("PUT", "/auth/password", changePassword)
