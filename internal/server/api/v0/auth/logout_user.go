package v0_auth

import (
	"strings"

	"github.com/autobutler-org/quark/pkg/util/authutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

// logoutUser godoc
// @Summary Logout
// @Description Invalidates the session the request carries, as a bearer token, the session cookie, or both, and clears the cookie
// @Tags auth
// @Produce json
// @Success 200 {object} object
// @Security BearerAuth
// @Router /auth/logout [post]
func logoutUser(c *gin.Context) *serverutil.Response {
	deps, ok := getQueries(c)
	if !ok || (*deps).Database() == nil {
		return serverutil.Ok()
	}

	// End every session the request names. A browser can send a leftover
	// bearer beside its cookie, and requireAuth may have let the request in on
	// either (#2156).
	queries := (*deps).Database().Queries
	if token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer "); ok && token != "" {
		_ = authutil.Logout(c.Request.Context(), queries, token)
	}
	if cookie, err := c.Cookie(sessionCookieName); err == nil && cookie != "" {
		_ = authutil.Logout(c.Request.Context(), queries, cookie)
	}

	clearSessionCookie(c)
	return serverutil.Ok().WithData(gin.H{"message": "logged out"})
}

var logoutUserRoute = serverutil.ApiRoute("POST", "/auth/logout", logoutUser)
