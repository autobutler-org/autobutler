package v0_auth

import (
	"net/http"
	"strconv"
	"time"

	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"

	"github.com/gin-gonic/gin"
)

const sessionCookieName = "session"
const sessionCookieMaxAge = int(30 * 24 * time.Hour / time.Second)

// isTLS returns true when the underlying connection or a trusted proxy indicates
// that the request arrived over HTTPS. Setting the Secure cookie flag on HTTP
// would prevent the cookie from being sent, so we only set it on TLS connections.
func isTLS(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	// Honour a reverse-proxy header (e.g. nginx → Quark over plain HTTP).
	return c.Request.Header.Get("X-Forwarded-Proto") == "https"
}

func setSessionCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(sessionCookieName, token, sessionCookieMaxAge, "/", "", isTLS(c), true)
}

func clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(sessionCookieName, "", -1, "/", "", isTLS(c), true)
}

func getQueries(c *gin.Context) (*deputil.Dependencies, bool) {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	return &deps, ok
}

// queryBool reads an optional boolean query parameter. An absent parameter is
// false; a present but unparseable one is rejected rather than silently
// defaulting, so a typo on a destructive endpoint cannot read as "no".
func queryBool(c *gin.Context, name string) (bool, bool) {
	raw, present := c.GetQuery(name)
	if !present || raw == "" {
		return false, true
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, false
	}
	return value, true
}
