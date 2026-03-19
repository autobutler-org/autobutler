package v1_auth

import (
	"net/http"
	"time"

	"github.com/autobutler-org/autobutler/pkg/util/authutil"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/instanceutil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

const sessionCookieName = "session"
const sessionCookieMaxAge = int(30 * 24 * time.Hour / time.Second)

func setSessionCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(sessionCookieName, token, sessionCookieMaxAge, "/", "", false, true)
}

func clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(sessionCookieName, "", -1, "/", "", false, true)
}

func getQueries(c *gin.Context) (*deputil.Dependencies, bool) {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	return &deps, ok
}

// AuthStatusJSON is the response body for GET /auth/status.
type AuthStatusJSON struct {
	// Setup indicates whether the initial account setup has been completed.
	Setup bool `json:"setup"`
	// InstanceID is the stable UUID assigned to this butler on first boot.
	// Clients use it to detect when a host address now points to a different
	// butler (e.g. a neighbour's device on the same LAN hostname).
	// Empty string when setup has not yet been completed.
	InstanceID string `json:"instanceId"`
}

// authStatus godoc
// @Summary Check auth setup status
// @Description Returns whether initial setup has been completed, and the butler's stable instance ID.
// @Tags auth
// @Produce json
// @Success 200 {object} AuthStatusJSON
// @Router /auth/status [get]
func authStatus(c *gin.Context) *serverutil.Response {
	deps, ok := getQueries(c)
	if !ok || (*deps).Database() == nil {
		return serverutil.Ok().WithData(AuthStatusJSON{Setup: false, InstanceID: ""})
	}

	complete, err := authutil.IsSetupComplete(c.Request.Context(), (*deps).Database().Queries)
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	// Include the instance ID so clients can detect when they've connected
	// to a different butler than expected (e.g. neighbour's butler at the
	// same LAN hostname — see issue #414).
	instanceID, _ := instanceutil.GetInstanceID(c.Request.Context(), (*deps).Database().Queries)

	return serverutil.Ok().WithData(AuthStatusJSON{Setup: complete, InstanceID: instanceID})
}

var authStatusRoute = serverutil.ApiRoute("GET", "/auth/status", authStatus)

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
func authSetup(c *gin.Context) *serverutil.Response {
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

	setSessionCookie(c, result.SessionToken)
	return serverutil.Ok().WithData(gin.H{
		"token":          result.SessionToken,
		"recoveryPhrase": result.RecoveryPhrase,
		"message":        "Setup complete. Store your recovery phrase somewhere safe — it will not be shown again.",
	})
}

var authSetupRoute = serverutil.ApiRoute("POST", "/auth/setup", authSetup)

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
func authLogin(c *gin.Context) *serverutil.Response {
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

	setSessionCookie(c, result.SessionToken)
	return serverutil.Ok().WithData(gin.H{"token": result.SessionToken})
}

var authLoginRoute = serverutil.ApiRoute("POST", "/auth/login", authLogin)

// authLogout godoc
// @Summary Logout
// @Description Invalidates the current session
// @Tags auth
// @Produce json
// @Success 200 {object} object
// @Router /auth/logout [post]
func authLogout(c *gin.Context) *serverutil.Response {
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

	clearSessionCookie(c)
	return serverutil.Ok().WithData(gin.H{"message": "logged out"})
}

var authLogoutRoute = serverutil.ApiRoute("POST", "/auth/logout", authLogout)

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
func authRecover(c *gin.Context) *serverutil.Response {
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

	setSessionCookie(c, result.SessionToken)
	return serverutil.Ok().WithData(gin.H{"token": result.SessionToken})
}

var authRecoverRoute = serverutil.ApiRoute("POST", "/auth/recover", authRecover)
