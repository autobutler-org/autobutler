package v0_vault

import (
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"

	"github.com/gin-gonic/gin"
)

// getVaultKey godoc
// @Summary Get the caller's wrapped vault key
// @Description Returns the caller's vault key, wrapped on the client under the login password and, when present, the recovery phrase. Byte fields are base64 and kdfParams is the client's own object. The Quark never opens it. 404 means the caller has no vault yet and the client should set one up with PUT /vault/key.
// @Tags vault
// @Produce json
// @Success 200 {object} vaultutil.Key
// @Failure 401 {object} serverutil.Response
// @Failure 404 {object} serverutil.Response "the caller has no vault yet"
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /vault/key [get]
func getVaultKey(c *gin.Context) *serverutil.Response {
	deps, userID, failed := callerContext(c)
	if failed != nil {
		return failed
	}
	result, err := vaultutil.GetKey(vaultutil.GetKeyParams{Ctx: c.Request.Context(), Queries: deps.Database().Queries, UserID: userID})
	if err != nil {
		return vaultError(err)
	}
	return serverutil.Ok().WithData(result.Key)
}

var getVaultKeyRoute = serverutil.ApiRoute("GET", "/vault/key", getVaultKey)
