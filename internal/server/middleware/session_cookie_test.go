package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/autobutler-org/quark/internal/db/dbtest"
	v0_auth "github.com/autobutler-org/quark/internal/server/api/v0/auth"
	"github.com/autobutler-org/quark/internal/server/middleware"
	"github.com/autobutler-org/quark/pkg/util/authutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/settingsutil"
	"github.com/autobutler-org/quark/pkg/vfs"
	"github.com/gin-gonic/gin"
)

// newCookieEngine is the middleware stack in front of the auth router and one
// state-changing route, with an account already set up. It returns that
// account's session token.
func newCookieEngine(t *testing.T) (*gin.Engine, string) {
	t.Helper()
	settingsutil.ResetForTesting(filepath.Join(t.TempDir(), "settings.json"))
	t.Cleanup(func() { settingsutil.ResetForTesting("") })
	database := dbtest.NewDB(t)
	result, err := authutil.Setup(context.Background(), authutil.SetupParams{
		Database: database, Files: vfs.NewMemVFS("files"),
		Username: "admin", AuthKey: dbtest.AuthKey("SecurePass1!"), SaltSecret: dbtest.SaltSecret,
	})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	deps := deputil.NewDependencies().WithDatabase(database)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	middleware.Use(engine, deps)
	t.Cleanup(deps.Background().Wait)
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	engine.GET("/api/v0/protected", ok)
	engine.POST("/api/v0/protected", ok)
	engine.GET("/api/v0/events", ok)
	serverutil.RegisterRouterWithGroup(engine.Group("/api/v0"), v0_auth.NewRouter())
	return engine, result.SessionToken
}

func sessionCookie(t *testing.T, w *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "session" {
			return cookie
		}
	}
	t.Fatalf("no session cookie in %v", w.Header())
	return nil
}

// TestSessionCookie_LoginToLogout walks the browser's session (#2156): login
// issues an HttpOnly, SameSite=Strict cookie carrying the token the body
// carries, that cookie alone authenticates, and logout with that cookie alone
// ends the session and clears the cookie.
func TestSessionCookie_LoginToLogout(t *testing.T) {
	engine, _ := newCookieEngine(t)

	body, _ := json.Marshal(map[string]string{"username": "admin", "authKey": dbtest.AuthKey("SecurePass1!")})
	login := httptest.NewRequest(http.MethodPost, "/api/v0/auth/login", bytes.NewReader(body))
	login.Header.Set("Content-Type", "application/json")
	w := doMiddlewareReq(engine, login)
	if w.Code != http.StatusOK {
		t.Fatalf("login = %d: %s", w.Code, w.Body.String())
	}
	var response struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil || response.Token == "" {
		t.Fatalf("login body %q: %v", w.Body.String(), err)
	}
	cookie := sessionCookie(t, w)
	if cookie.Value != response.Token {
		t.Error("the cookie must carry the session token the body carries")
	}
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/" || cookie.Domain != "" {
		t.Errorf("cookie = %+v, want HttpOnly, SameSite=Strict, Path=/ and host-only", cookie)
	}
	if cookie.Secure {
		t.Error("a cookie marked Secure on plain HTTP is never sent back")
	}
	if cookie.MaxAge != int(authutil.SessionMaxLifetime.Seconds()) {
		t.Errorf("Max-Age = %d, want the session's longest life: a renewed session must not lose its cookie first", cookie.MaxAge)
	}

	withCookie := func(method, path string) *http.Request {
		req := httptest.NewRequest(method, path, nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: cookie.Value})
		return req
	}
	if w := doMiddlewareReq(engine, withCookie(http.MethodGet, "/api/v0/protected")); w.Code != http.StatusOK {
		t.Fatalf("cookie alone = %d, want 200", w.Code)
	}

	w = doMiddlewareReq(engine, withCookie(http.MethodPost, "/api/v0/auth/logout"))
	if w.Code != http.StatusOK {
		t.Fatalf("logout = %d: %s", w.Code, w.Body.String())
	}
	if cleared := sessionCookie(t, w); cleared.Value != "" || cleared.MaxAge >= 0 || !cleared.HttpOnly {
		t.Errorf("logout cookie = %+v, want an expired, empty, HttpOnly one", cleared)
	}
	if w := doMiddlewareReq(engine, withCookie(http.MethodGet, "/api/v0/protected")); w.Code != http.StatusUnauthorized {
		t.Errorf("cookie after logout = %d, want 401", w.Code)
	}
}

// TestSessionCookie_SecureOverTLS shows the cookie is marked Secure when the
// sign-in arrived over TLS.
func TestSessionCookie_SecureOverTLS(t *testing.T) {
	engine, _ := newCookieEngine(t)

	body, _ := json.Marshal(map[string]string{"username": "admin", "authKey": dbtest.AuthKey("SecurePass1!")})
	login := httptest.NewRequest(http.MethodPost, "https://quark.local/api/v0/auth/login", bytes.NewReader(body))
	login.Header.Set("Content-Type", "application/json")
	w := doMiddlewareReq(engine, login)
	if w.Code != http.StatusOK {
		t.Fatalf("login = %d: %s", w.Code, w.Body.String())
	}
	if !sessionCookie(t, w).Secure {
		t.Error("a cookie issued over TLS must be Secure")
	}
}

// TestLogout_EndsTheCookieSessionBesideAStaleBearer covers a browser that
// sends a leftover Authorization header with its cookie: the cookie's session
// is the one that authenticated, so it is the one that has to end.
func TestLogout_EndsTheCookieSessionBesideAStaleBearer(t *testing.T) {
	engine, token := newCookieEngine(t)

	logout := httptest.NewRequest(http.MethodPost, "/api/v0/auth/logout", nil)
	logout.Header.Set("Authorization", "Bearer stale")
	logout.AddCookie(&http.Cookie{Name: "session", Value: token})
	if w := doMiddlewareReq(engine, logout); w.Code != http.StatusOK {
		t.Fatalf("logout = %d: %s", w.Code, w.Body.String())
	}

	after := httptest.NewRequest(http.MethodGet, "/api/v0/protected", nil)
	after.Header.Set("Authorization", "Bearer "+token)
	if w := doMiddlewareReq(engine, after); w.Code != http.StatusUnauthorized {
		t.Errorf("session after logout = %d, want 401", w.Code)
	}
}

// TestRequireAuth_CookieRefusesCrossOrigin is the CSRF rule (#2156). A cookie
// rides along on any request the browser makes, so a state-changing request or
// a WebSocket upgrade it authenticates must come from the Quark's own origin.
// A credential the page had to hold, a bearer or ?token=, is proof enough on
// its own, and a client that sends no Origin is not a browser.
func TestRequireAuth_CookieRefusesCrossOrigin(t *testing.T) {
	engine, token := newCookieEngine(t)

	// httptest.NewRequest addresses example.com.
	const sameOrigin = "http://example.com"
	cases := []struct {
		name    string
		method  string
		path    string
		headers map[string]string
		cookie  bool
		want    int
	}{
		{name: "same origin", method: http.MethodPost, cookie: true, headers: map[string]string{"Origin": sameOrigin}, want: http.StatusOK},
		{name: "browser says same-origin", method: http.MethodPost, cookie: true, headers: map[string]string{"Sec-Fetch-Site": "same-origin", "Origin": sameOrigin}, want: http.StatusOK},
		{name: "no Origin: not a browser", method: http.MethodPost, cookie: true, want: http.StatusOK},
		{name: "another site", method: http.MethodPost, cookie: true, headers: map[string]string{"Origin": "https://evil.example"}, want: http.StatusForbidden},
		{name: "browser says cross-site", method: http.MethodPost, cookie: true, headers: map[string]string{"Sec-Fetch-Site": "cross-site"}, want: http.StatusForbidden},
		// SameSite=Strict lets these two through: a site is not an origin.
		{name: "same site, another port", method: http.MethodPost, cookie: true, headers: map[string]string{"Origin": "http://example.com:8081"}, want: http.StatusForbidden},
		{name: "browser says same-site", method: http.MethodPost, cookie: true, headers: map[string]string{"Sec-Fetch-Site": "same-site"}, want: http.StatusForbidden},
		{name: "cross-origin read", method: http.MethodGet, cookie: true, headers: map[string]string{"Origin": "https://evil.example"}, want: http.StatusOK},
		{name: "cross-origin WebSocket", method: http.MethodGet, path: "/api/v0/events", cookie: true, headers: map[string]string{"Origin": "https://evil.example", "Upgrade": "websocket"}, want: http.StatusForbidden},
		{name: "same-origin WebSocket", method: http.MethodGet, path: "/api/v0/events", cookie: true, headers: map[string]string{"Origin": sameOrigin, "Upgrade": "websocket"}, want: http.StatusOK},
		{name: "cross-origin WebSocket with its own token", method: http.MethodGet, path: "/api/v0/events?token=" + token, cookie: true, headers: map[string]string{"Origin": "http://example.com:8081", "Upgrade": "websocket"}, want: http.StatusOK},
		{name: "cross-origin bearer", method: http.MethodPost, headers: map[string]string{"Origin": "chrome-extension://abcdefghijklmnop", "Authorization": "Bearer " + token}, want: http.StatusOK},
		{name: "cross-origin bearer beside the cookie", method: http.MethodPost, cookie: true, headers: map[string]string{"Origin": "http://example.com:8081", "Authorization": "Bearer " + token}, want: http.StatusOK},
		{name: "cross-origin stale bearer beside the cookie", method: http.MethodPost, cookie: true, headers: map[string]string{"Origin": "https://evil.example", "Authorization": "Bearer stale"}, want: http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.path
			if path == "" {
				path = "/api/v0/protected"
			}
			req := httptest.NewRequest(tc.method, path, nil)
			for name, value := range tc.headers {
				req.Header.Set(name, value)
			}
			if tc.cookie {
				req.AddCookie(&http.Cookie{Name: "session", Value: token})
			}
			if w := doMiddlewareReq(engine, req); w.Code != tc.want {
				t.Errorf("status = %d, want %d: %s", w.Code, tc.want, w.Body.String())
			}
		})
	}
}
