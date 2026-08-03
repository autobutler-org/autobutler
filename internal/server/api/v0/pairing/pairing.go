package v0_pairing

import (
	"fmt"
	"net/http"

	qrcode "github.com/skip2/go-qrcode"

	"github.com/autobutler-org/autobutler/pkg/util/pairingutil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

// getPairingQR godoc
// @Summary Generate mobile pairing QR code
// @Description Returns a QR code PNG (or the raw pairing URL as JSON) encoding a
// @Description short-lived token. Scan with a phone to open the AutoButler Flutter
// @Description app pre-configured with this butler's LAN address.
// @Tags pairing
// @Produce image/png
// @Produce application/json
// @Param format query string false "Response format: 'png' (default) or 'json'"
// @Param size   query int    false "QR image size in pixels (default 256)"
// @Success 200 {file}   binary "QR PNG image"
// @Success 200 {object} object "JSON: {token, url, expiresIn}"
// @Failure 500 {object} serverutil.Response
// @Router /pairing/qr [get]
func getPairingQR(c *gin.Context) *serverutil.Response {
	port := serverutil.ServingPort()
	tls := serverutil.ServingTLS()
	scheme := "http"
	if tls {
		scheme = "https"
	}

	lanAddr := pairingutil.LANAddress(port)
	token, err := pairingutil.IssuePairingToken(lanAddr, scheme)
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	// The pairing URL is opened on the phone. It deep-links into the
	// Flutter app (or shows a landing page if the app isn't installed).
	// Format: autobutler://pair?token=<jwt>
	// Fallback web URL: https://<lanAddr>/mobile?token=<jwt>
	deepLink := fmt.Sprintf("autobutler://pair?token=%s", token)
	webURL := fmt.Sprintf("%s://%s/mobile?token=%s", scheme, lanAddr, token)

	format := c.DefaultQuery("format", "png")
	if format == "json" {
		return serverutil.Ok().WithData(gin.H{
			"token":     token,
			"deepLink":  deepLink,
			"webURL":    webURL,
			"expiresIn": "10m",
		})
	}

	size := 256
	if s := c.Query("size"); s != "" {
		fmt.Sscanf(s, "%d", &size)
		if size < 64 {
			size = 64
		}
		if size > 1024 {
			size = 1024
		}
	}

	// Encode the deep-link URI; fall back to web URL for camera apps that
	// don't know about the autobutler:// scheme.
	// We embed both: the QR value is the deep-link, and we add the web URL
	// as a comment so any QR reader can fall back.
	pngBytes, err := qrcode.Encode(deepLink, qrcode.Medium, size)
	if err != nil {
		return serverutil.InternalServerError(fmt.Errorf("qr encode: %w", err))
	}

	c.Header("Content-Type", "image/png")
	c.Header("Cache-Control", "no-store") // tokens must not be cached
	c.Header("X-Pairing-Web-URL", webURL) // for JS callers that want the fallback URL
	c.Header("X-Pairing-Expires-In", "600") // seconds
	c.Data(http.StatusOK, "image/png", pngBytes)
	return nil
}

var getPairingQRRoute = serverutil.ApiRoute("GET", "/pairing/qr", getPairingQR)

// validatePairingToken godoc
// @Summary Validate a pairing token
// @Description Called by the mobile app after scanning the QR code to verify
// @Description the token is valid and extract the butler's address.
// @Tags pairing
// @Accept json
// @Produce json
// @Param body body object true "{token}"
// @Success 200 {object} object "{addr, scheme, valid}"
// @Failure 400 {object} serverutil.Response
// @Router /pairing/validate [post]
func validatePairingToken(c *gin.Context) *serverutil.Response {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(err)
	}

	claims, err := pairingutil.ValidatePairingToken(req.Token)
	if err != nil {
		return serverutil.NewResponse().
			WithStatusCode(http.StatusUnauthorized).
			WithError(fmt.Errorf("invalid or expired pairing token"))
	}

	return serverutil.Ok().WithData(gin.H{
		"addr":   claims.ButlerAddr,
		"scheme": claims.Scheme,
		"valid":  true,
	})
}

var validatePairingTokenRoute = serverutil.ApiRoute("POST", "/pairing/validate", validatePairingToken)

type router struct{}

// NewRouter returns the pairing router.
func NewRouter() serverutil.Router { return &router{} }

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		getPairingQRRoute,
		validatePairingTokenRoute,
	}
}
