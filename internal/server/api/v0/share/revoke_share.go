package v0_share

import (
	"errors"
	"fmt"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

// revokeShare godoc
// @Summary Revoke a share link
// @Description Delete a share link by its token hash. Only the owner can revoke.
// @Tags share
// @Param tokenHash path string true "Token hash returned by GET /share"
// @Success 200 {object} serverutil.Response
// @Failure 401 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Router /share/{tokenHash} [delete]
func revokeShare(c *gin.Context) *serverutil.Response {
	tokenHash := c.Param("tokenHash")
	if tokenHash == "" {
		return serverutil.BadRequest(errors.New("tokenHash path parameter is required"))
	}

	username, ok := ctxutil.Get[string](c, "username")
	if !ok || username == "" {
		return serverutil.Unauthorized(errors.New("authentication required"))
	}

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	database := deps.Database()
	if database == nil {
		return serverutil.InternalServerError(errors.New("database unavailable"))
	}

	user, err := database.Queries.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		return serverutil.InternalServerError(fmt.Errorf("resolve user: %w", err))
	}

	if err := database.Queries.DeleteShareLink(c.Request.Context(),
		dbDeleteShareLinkParams(tokenHash, user.ID)); err != nil {
		return serverutil.InternalServerError(fmt.Errorf("revoke share link: %w", err))
	}

	return serverutil.Ok()
}

var revokeShareRoute = serverutil.ApiRoute("DELETE", "/share/:tokenHash", revokeShare)
