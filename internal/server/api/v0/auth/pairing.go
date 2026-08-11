package v0_auth

import (
	"fmt"
	"net/http"

	"github.com/autobutler-org/autobutler/pkg/util/authutil"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	qrcode "github.com/skip2/go-qrcode"

	"github.com/gin-gonic/gin"
)

// pairingURL builds the pairing exchange URL that gets embedded in the QR code.
// The phone scans this URL and POSTs it to /api/v0/auth/pair to receive a session.
func pairingURL(c *gin.Context, token string) string {
	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s/api/v0/auth/pair?token=%s", scheme, c.Request.Host, token)
}

// generatePairingCode godoc
// @Summary Generate a QR pairing code
// @Description Creates a short-lived (10 min) single-use pairing token and returns its URL encoded as a PNG QR code. The admin scans or displays this from /mobile to onboard a new device without typing credentials.
// @Tags auth
// @Produce image/png
// @Param size query int false "QR image size in pixels (default 256)"
// @Success 200 {file} file "PNG QR code image"
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /auth/pairing [get]
func generatePairingCode(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok || deps.Database() == nil {
		return serverutil.InternalServerError(nil)
	}

	user, err := deps.Database().Queries.GetFirstUser(c.Request.Context())
	if err != nil {
		return serverutil.NewResponse().WithStatusCode(http.StatusUnauthorized).WithError(err)
	}

	token, err := authutil.CreatePairingToken(c.Request.Context(), deps.Database().Queries, user.ID)
	if err != nil {
		return serverutil.InternalServerError(fmt.Errorf("create pairing token: %w", err))
	}

	url := pairingURL(c, token)

	size := 256
	if raw := c.Query("size"); raw != "" {
		var s int
		if _, err := fmt.Sscanf(raw, "%d", &s); err == nil && s >= 64 && s <= 1024 {
			size = s
		}
	}

	png, err := qrcode.Encode(url, qrcode.High, size)
	if err != nil {
		return serverutil.InternalServerError(fmt.Errorf("encode qr code: %w", err))
	}

	c.Data(http.StatusOK, "image/png", png)
	return nil
}

var generatePairingCodeRoute = serverutil.ApiRoute("GET", "/auth/pairing", generatePairingCode)

// exchangePairingToken godoc
// @Summary Exchange a pairing token for a session
// @Description Consumes a single-use pairing token (from scanning the /mobile QR code) and returns a session token. Rate-limited.
// @Tags auth
// @Produce json
// @Param token query string true "Raw pairing token from the QR code URL"
// @Success 200 {object} object{token=string}
// @Failure 400 {object} serverutil.Response
// @Failure 401 {object} serverutil.Response
// @Router /auth/pair [post]
func exchangePairingToken(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok || deps.Database() == nil {
		return serverutil.InternalServerError(nil)
	}

	rawToken := c.Query("token")
	if rawToken == "" {
		// Also accept token in JSON body for programmatic callers.
		var req struct {
			Token string `json:"token"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			rawToken = req.Token
		}
	}
	if rawToken == "" {
		return serverutil.BadRequest(fmt.Errorf("token is required"))
	}

	result, err := authutil.ConsumePairingToken(c.Request.Context(), deps.Database().Queries, rawToken)
	if err != nil {
		return serverutil.NewResponse().WithStatusCode(http.StatusUnauthorized).WithError(err)
	}

	setSessionCookie(c, result.SessionToken)
	return serverutil.Ok().WithData(gin.H{"token": result.SessionToken})
}

var exchangePairingTokenRoute = serverutil.ApiRoute("POST", "/auth/pair", exchangePairingToken)
