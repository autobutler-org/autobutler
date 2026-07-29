package v0_auth

import (
	"net/http"
	"time"

	"github.com/autobutler-org/autobutler/pkg/util/authutil"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

const sessionCookieName = "session"
const sessionCookieMaxAge = int(30 * 24 * time.Hour / time.Second)

// setSessionCookie writes the session cookie. The Secure flag is set when TLS
// is active (i.e. insecure == false) so the cookie is only ever sent over HTTPS.
func (r *router) setSessionCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(sessionCookieName, token, sessionCookieMaxAge, "/", "", !r.insecure, true)
}

func (r *router) clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(sessionCookieName, "", -1, "/", "", !r.insecure, true)
}

func getQueries(c *gin.Context) (*deputil.Dependencies, bool) {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	return &deps, ok
}

// authStatus godoc
// @Summary Check auth setup status
// @Description Returns whether initial setup has been completed
// @Tags auth
// @Produce json
// @Success 200 {object} object
// @Router /auth/status [get]
func authStatus(c *gin.Context) *serverutil.Response {
	deps, ok := getQueries(c)
	if !ok || (*deps).Database() == nil {
		return serverutil.Ok().WithData(gin.H{"setup": false})
	}

	complete, err := authutil.IsSetupComplete(c.Request.Context(), (*deps).Database().Queries)
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	return serverutil.Ok().WithData(gin.H{"setup": complete})
}

// authSetup godoc
// @Summary First-boot user setup
// @Description Creates the owner account. Can only be called once.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body object true "{username, password}"
// @Success 200 {object} object
// @Failure 400 {object} serverutil.Response
// @Router /auth/setup [post]
func (r *router) authSetup(c *gin.Context) *serverutil.Response {
	deps, ok := getQueries(c)
	if !ok || (*deps).Database() == nil {
		return serverutil.InternalServerError(nil)
	}

	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(err)
	}

	result, err := authutil.Setup(c.Request.Context(), (*deps).Database().Queries, authutil.SetupParams{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		return serverutil.BadRequest(err)
	}

	r.setSessionCookie(c, result.SessionToken)
	return serverutil.Ok().WithData(gin.H{
		"token":          result.SessionToken,
		"recoveryPhrase": result.RecoveryPhrase,
		"message":        "Setup complete. Store your recovery phrase somewhere safe — it will not be shown again.",
	})
}

// authLogin godoc
// @Summary Login
// @Description Authenticates with username and password, returns a session token
// @Tags auth
// @Accept json
// @Produce json
// @Param body body object true "{username, password}"
// @Success 200 {object} object
// @Failure 401 {object} serverutil.Response
// @Router /auth/login [post]
func (r *router) authLogin(c *gin.Context) *serverutil.Response {
	deps, ok := getQueries(c)
	if !ok || (*deps).Database() == nil {
		return serverutil.InternalServerError(nil)
	}

	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(err)
	}

	result, err := authutil.Login(c.Request.Context(), (*deps).Database().Queries, authutil.LoginParams{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		return serverutil.NewResponse().WithStatusCode(http.StatusUnauthorized).WithError(err)
	}

	r.setSessionCookie(c, result.SessionToken)
	return serverutil.Ok().WithData(gin.H{"token": result.SessionToken})
}

// authLogout godoc
// @Summary Logout
// @Description Invalidates the current session
// @Tags auth
// @Produce json
// @Success 200 {object} object
// @Router /auth/logout [post]
func (r *router) authLogout(c *gin.Context) *serverutil.Response {
	deps, ok := getQueries(c)
	if !ok || (*deps).Database() == nil {
		return serverutil.Ok()
	}

	token := ""
	if auth := c.GetHeader("Authorization"); len(auth) > 7 && auth[:7] == "Bearer " {
		token = auth[7:]
	} else if cookie, err := c.Cookie(sessionCookieName); err == nil {
		token = cookie
	}

	if token != "" {
		_ = authutil.Logout(c.Request.Context(), (*deps).Database().Queries, token)
	}

	r.clearSessionCookie(c)
	return serverutil.Ok().WithData(gin.H{"message": "logged out"})
}

// authRecover godoc
// @Summary Recover account
// @Description Resets password using recovery phrase
// @Tags auth
// @Accept json
// @Produce json
// @Param body body object true "{recoveryPhrase, newPassword}"
// @Success 200 {object} object
// @Failure 400 {object} serverutil.Response
// @Router /auth/recover [post]
func (r *router) authRecover(c *gin.Context) *serverutil.Response {
	deps, ok := getQueries(c)
	if !ok || (*deps).Database() == nil {
		return serverutil.InternalServerError(nil)
	}

	var req struct {
		RecoveryPhrase string `json:"recoveryPhrase" binding:"required"`
		NewPassword    string `json:"newPassword" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(err)
	}

	result, err := authutil.Recover(c.Request.Context(), (*deps).Database().Queries, authutil.RecoverParams{
		RecoveryPhrase: req.RecoveryPhrase,
		NewPassword:    req.NewPassword,
	})
	if err != nil {
		return serverutil.BadRequest(err)
	}

	r.setSessionCookie(c, result.SessionToken)
	return serverutil.Ok().WithData(gin.H{"token": result.SessionToken})
}
