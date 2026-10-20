package v0_vault

import (
	"errors"
	"net/http"

	"github.com/autobutler-org/quark/pkg/util/authutil"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"

	"github.com/gin-gonic/gin"
)

// createVaultKey godoc
// @Summary Set up the caller's vault
// @Description Re-confirms the caller with password, which carries the auth key of the login password, exactly as PUT /auth/recovery-key takes it; a wrong or missing key is a 403, a raw password is a 426, and nothing is written. The vault key's password wrap is opened, and re-wrapped on every password change, with the login password, so a vault wrapped under any other password would be lost at the next change. Then stores the caller's first vault key, a random key the client made and wrapped under the login password (wrappedByPassword, at most 512 bytes, with its 16-byte Argon2id salt saltPw) and optionally the recovery phrase (wrappedByPhrase and saltRp, both or neither), with the client's kdfParams object. Byte fields are base64 and the body is at most 8 KiB. The key is never replaced here: an account that already has one gets 409, and its wraps change only beside the credential that wraps them, through vaultKey on PUT /auth/password, PUT /auth/recovery-key and POST /auth/recover. Publishes vault_changed to the caller's sessions. Shares the sign-in rate limit.
// @Tags vault
// @Accept json
// @Produce json
// @Param body body createVaultKeyBody true "The caller's auth key and the wrapped vault key; createdAt and updatedAt are ignored"
// @Success 201 {object} vaultutil.Key
// @Failure 400 {object} serverutil.Response "a key of the wrong size or shape"
// @Failure 401 {object} serverutil.Response
// @Failure 403 {object} serverutil.Response "password is not the caller's auth key"
// @Failure 409 {object} serverutil.Response "the caller already has a vault"
// @Failure 413 {object} serverutil.Response "a body over 8 KiB"
// @Failure 426 {object} serverutil.Response "password is a raw password: the app is too old"
// @Failure 429 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /vault/key [put]
func createVaultKey(c *gin.Context) *serverutil.Response {
	deps, userID, failed := callerContext(c)
	if failed != nil {
		return failed
	}
	var body createVaultKeyBody
	if failed := bindBody(c, vaultutil.MaxKeyRequestBytes, &body, vaultutil.ErrInvalidKey); failed != nil {
		return failed
	}
	if err := vaultutil.ValidateKey(body.Key); err != nil {
		return vaultError(err)
	}
	// Checked before anything is written: a key wrapped under anything but the
	// login password would not open after the next password change.
	username, _ := ctxutil.Get[string](c, "username")
	_, err := authutil.VerifyPassword(c.Request.Context(), authutil.VerifyPasswordParams{
		Queries:  deps.Database().Queries,
		Username: username,
		Password: body.Password,
	})
	switch {
	case errors.Is(err, authutil.ErrIncorrectPassword):
		return serverutil.NewResponse().WithStatusCode(http.StatusForbidden).WithError(err)
	case errors.Is(err, authutil.ErrUserNotFound):
		return serverutil.Unauthorized(errors.New("not authenticated"))
	case err != nil:
		// ErrAppTooOld is a 426 HttpError, whatever status this names.
		return serverutil.InternalServerError(err)
	}
	result, err := vaultutil.CreateKey(vaultutil.CreateKeyParams{Ctx: c.Request.Context(), Queries: deps.Database().Queries, UserID: userID, Key: body.Key})
	if err != nil {
		return vaultError(err)
	}
	vaultutil.PublishChanged(vaultutil.PublishChangedParams{EventBus: deps.EventBus(), UserID: userID})
	return serverutil.NewResponse().WithStatusCode(http.StatusCreated).WithData(result.Key)
}

var createVaultKeyRoute = serverutil.ApiRoute("PUT", "/vault/key", createVaultKey)
