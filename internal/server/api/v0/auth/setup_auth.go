package v0_auth

import (
	"github.com/autobutler-org/quark/pkg/util/authutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

// setupAuth godoc
// @Summary First-boot user setup
// @Description Creates the owner account. Can only be called once.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body object true "{username, password}"
// @Success 200 {object} object
// @Failure 400 {object} serverutil.Response
// @Router /auth/setup [post]
func setupAuth(c *gin.Context) *serverutil.Response {
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

var setupAuthRoute = serverutil.ApiRoute("POST", "/auth/setup", setupAuth)
