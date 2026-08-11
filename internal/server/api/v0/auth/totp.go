package v0_auth

import (
	"fmt"
	"net/http"

	"github.com/autobutler-org/autobutler/pkg/util/authutil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// totpEnroll godoc
// @Summary Begin TOTP enrollment
// @Description Generates a new TOTP secret and stores it as pending. Returns the secret and an otpauth:// URL for QR code display. The secret is not activated until confirmed with /auth/totp/confirm.
// @Tags auth
// @Produce json
// @Success 200 {object} object{secret=string,otpURL=string}
// @Failure 401 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /auth/totp/enroll [post]
func totpEnroll(c *gin.Context) *serverutil.Response {
	deps, ok := getQueries(c)
	if !ok || (*deps).Database() == nil {
		return serverutil.InternalServerError(nil)
	}

	user, err := (*deps).Database().Queries.GetFirstUser(c.Request.Context())
	if err != nil {
		return serverutil.NewResponse().WithStatusCode(http.StatusUnauthorized).WithError(err)
	}

	result, err := authutil.TOTPEnroll(c.Request.Context(), (*deps).Database().Queries, user.ID, user.Username)
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	return serverutil.Ok().WithData(gin.H{
		"secret": result.Secret,
		"otpURL": result.OTPURL,
	})
}

var totpEnrollRoute = serverutil.ApiRoute("POST", "/auth/totp/enroll", totpEnroll)

// totpConfirm godoc
// @Summary Confirm TOTP enrollment
// @Description Validates the TOTP code against the pending secret and activates 2FA. Must be called after /auth/totp/enroll.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body object true "{code}"
// @Success 200 {object} object{message=string}
// @Failure 400 {object} serverutil.Response
// @Failure 401 {object} serverutil.Response
// @Security BearerAuth
// @Router /auth/totp/confirm [post]
func totpConfirm(c *gin.Context) *serverutil.Response {
	deps, ok := getQueries(c)
	if !ok || (*deps).Database() == nil {
		return serverutil.InternalServerError(nil)
	}

	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(err)
	}

	user, err := (*deps).Database().Queries.GetFirstUser(c.Request.Context())
	if err != nil {
		return serverutil.NewResponse().WithStatusCode(http.StatusUnauthorized).WithError(err)
	}

	if err := authutil.TOTPConfirm(c.Request.Context(), (*deps).Database().Queries, user.ID, req.Code); err != nil {
		return serverutil.BadRequest(err)
	}

	return serverutil.Ok().WithData(gin.H{"message": "2FA enabled"})
}

var totpConfirmRoute = serverutil.ApiRoute("POST", "/auth/totp/confirm", totpConfirm)

// totpDisable godoc
// @Summary Disable 2FA
// @Description Disables TOTP 2FA for the account. Requires the current password as confirmation.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body object true "{password}"
// @Success 200 {object} object{message=string}
// @Failure 400 {object} serverutil.Response
// @Failure 401 {object} serverutil.Response
// @Security BearerAuth
// @Router /auth/totp [delete]
func totpDisable(c *gin.Context) *serverutil.Response {
	deps, ok := getQueries(c)
	if !ok || (*deps).Database() == nil {
		return serverutil.InternalServerError(nil)
	}

	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(err)
	}

	user, err := (*deps).Database().Queries.GetFirstUser(c.Request.Context())
	if err != nil {
		return serverutil.NewResponse().WithStatusCode(http.StatusUnauthorized).WithError(err)
	}

	if !authutil.CheckPassword(req.Password, user.PasswordHash) {
		return serverutil.NewResponse().WithStatusCode(http.StatusUnauthorized).WithError(
			fmt.Errorf("invalid password"),
		)
	}

	if err := authutil.TOTPDisable(c.Request.Context(), (*deps).Database().Queries, user.ID); err != nil {
		return serverutil.InternalServerError(err)
	}

	return serverutil.Ok().WithData(gin.H{"message": "2FA disabled"})
}

var totpDisableRoute = serverutil.ApiRoute("DELETE", "/auth/totp", totpDisable)

// totpVerify godoc
// @Summary Complete login with 2FA code
// @Description Consumes a challengeToken (from /auth/login when requires2FA=true) and a TOTP code, returns a full session token on success.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body object true "{challengeToken, code}"
// @Success 200 {object} object{token=string}
// @Failure 400 {object} serverutil.Response
// @Failure 401 {object} serverutil.Response
// @Router /auth/totp/verify [post]
func totpVerify(c *gin.Context) *serverutil.Response {
	deps, ok := getQueries(c)
	if !ok || (*deps).Database() == nil {
		return serverutil.InternalServerError(nil)
	}

	var req struct {
		ChallengeToken string `json:"challengeToken" binding:"required"`
		Code           string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(err)
	}

	result, err := authutil.VerifyTOTPChallenge(
		c.Request.Context(),
		(*deps).Database().Queries,
		req.ChallengeToken,
		req.Code,
	)
	if err != nil {
		return serverutil.NewResponse().WithStatusCode(http.StatusUnauthorized).WithError(err)
	}

	setSessionCookie(c, result.SessionToken)
	return serverutil.Ok().WithData(gin.H{"token": result.SessionToken})
}

var totpVerifyRoute = serverutil.ApiRoute("POST", "/auth/totp/verify", totpVerify)

// totpStatus godoc
// @Summary Get 2FA status
// @Description Returns whether 2FA is currently enabled for the account.
// @Tags auth
// @Produce json
// @Success 200 {object} object{enabled=bool}
// @Security BearerAuth
// @Router /auth/totp [get]
func totpStatus(c *gin.Context) *serverutil.Response {
	deps, ok := getQueries(c)
	if !ok || (*deps).Database() == nil {
		return serverutil.InternalServerError(nil)
	}

	user, err := (*deps).Database().Queries.GetFirstUser(c.Request.Context())
	if err != nil {
		return serverutil.NewResponse().WithStatusCode(http.StatusUnauthorized).WithError(err)
	}

	enabled, err := authutil.TOTPIsEnabled(c.Request.Context(), (*deps).Database().Queries, user.ID)
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	return serverutil.Ok().WithData(gin.H{"enabled": enabled})
}

var totpStatusRoute = serverutil.ApiRoute("GET", "/auth/totp", totpStatus)
