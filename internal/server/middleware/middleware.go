package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/autobutler-org/autobutler/internal/db"
	v1_webdav "github.com/autobutler-org/autobutler/internal/server/api/v1/webdav"
	"github.com/autobutler-org/autobutler/pkg/util/authutil"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/ratelimitutil"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// authRateLimiter protects auth endpoints (login, setup, recover) from
// brute-force attacks. Shared across all requests — 5 req/s per IP, burst 10.
var authRateLimiter = ratelimitutil.New()

// vaultRateLimiter protects /vault/unlock from master-password brute-force.
// Tighter than the general auth limiter: 1 req/2s per IP, burst 5.
// After exhausting the burst, the steady-state cap is 0.5 req/s (one every 2s).
// Combined with Argon2id (~300 ms/attempt), sustained guessing is limited to
// ≈ 30 attempts/minute per IP — well below what any offline attack would need.
var vaultRateLimiter = ratelimitutil.NewWithRate(0.5, 5)

// authRateLimitedPaths are the API paths that require rate limiting.
var authRateLimitedPaths = map[string]bool{
	"/api/v1/auth/login":           true,
	"/api/v1/auth/setup":           true,
	"/api/v1/auth/recover":         true,
	"/api/v1/storage/devices/role": true,
}

// vaultRateLimitedPaths are the API paths that use the stricter vault limiter.
var vaultRateLimitedPaths = map[string]bool{
	"/api/v1/vault/unlock": true,
}

// rateLimit is a middleware that enforces per-IP rate limiting on auth endpoints.
func rateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := ratelimitutil.ExtractIP(c.ClientIP())
		path := c.Request.URL.Path
		if vaultRateLimitedPaths[path] {
			if !vaultRateLimiter.Allow(ip) {
				c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, please slow down"})
				c.Abort()
				return
			}
			c.Next()
			return
		}
		if !authRateLimitedPaths[path] {
			c.Next()
			return
		}
		if !authRateLimiter.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, please slow down"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// authExemptPaths are API paths that don't require a valid session.
var authExemptPaths = map[string]bool{
	"/api/v1/auth/setup":   true,
	"/api/v1/auth/login":   true,
	"/api/v1/auth/recover": true,
	"/api/v1/auth/status":  true,
}

func inject(deps deputil.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		c = ctxutil.With(c, "deps", deps)
		c.Next()
	}
}

// trackDevice records the client IP and User-Agent in connected_devices.
// Runs asynchronously so it never blocks the request.
func trackDevice(deps deputil.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if deps.Database() == nil {
			return
		}
		ip := c.ClientIP()
		ua := c.Request.UserAgent()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if _, err := deps.Database().Queries.UpsertConnectedDevice(
				ctx,
				db.UpsertConnectedDeviceParams{IpAddress: ip, UserAgent: ua},
			); err != nil {
				slog.Debug("trackDevice: upsert failed", "err", err)
			}
		}()
	}
}

// requireAuth validates the request using a fallthrough chain of auth methods.
// Each method is tried in order; if one fails, the next is attempted. Only if
// ALL methods fail does the request get a 401.
//
// Precedence (highest to lowest):
//  1. Bearer token (Authorization: Bearer <token>)
//  2. Session cookie
//  3. Query parameter (?token=)
//  4. HTTP Basic Auth (Authorization: Basic <base64>)
//
// Exempt paths (setup, login, recover, status) are always allowed through.
// If no users have been set up yet, all requests are allowed through (first-boot).
//
// setupDone caches whether initial setup has been completed. Once true it is
// never re-checked, avoiding a DB query on every request.
func requireAuth(deps deputil.Dependencies) gin.HandlerFunc {
	var setupDone atomic.Bool
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Static assets and the Flutter web app don't need auth — the client-side
		// AuthGate handles the login flow. Only /api/ and /dav/ routes require a session.
		if !strings.HasPrefix(path, "/api/") && !v1_webdav.IsWebDAVPath(path) {
			c.Next()
			return
		}

		if authExemptPaths[path] {
			c.Next()
			return
		}

		db := deps.Database()
		if db == nil {
			c.Next()
			return
		}

		// Check if setup is complete. Once confirmed, cache the result so we
		// don't query the DB on every subsequent request.
		if !setupDone.Load() {
			complete, err := authutil.IsSetupComplete(c.Request.Context(), db.Queries)
			if err != nil || !complete {
				c.Next()
				return
			}
			setupDone.Store(true)
		}

		ctx := c.Request.Context()

		// Collect all candidate tokens from headers/cookie/query.
		// Try each in order — fall through on failure.
		var tokens []string
		if auth := c.GetHeader("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			tokens = append(tokens, strings.TrimPrefix(auth, "Bearer "))
		}
		if cookie, err := c.Cookie("session"); err == nil && cookie != "" {
			tokens = append(tokens, cookie)
		}
		if q := c.Query("token"); q != "" {
			tokens = append(tokens, q)
		}

		for _, t := range tokens {
			username, err := authutil.ValidateSession(ctx, db.Queries, t)
			if err == nil {
				c = ctxutil.With(c, "username", username)
				c.Next()
				return
			}
		}

		// Fall back to HTTP Basic Auth.
		if username, password, ok := c.Request.BasicAuth(); ok {
			validUser, err := authutil.ValidateBasicAuth(ctx, db.Queries, username, password)
			if err == nil {
				c = ctxutil.With(c, "username", validUser)
				c.Next()
				return
			}
		}

		// All methods exhausted.
		// WebDAV clients need the WWW-Authenticate header to prompt for credentials.
		if v1_webdav.IsWebDAVPath(path) {
			c.Header("WWW-Authenticate", `Basic realm="AutoButler"`)
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		c.Abort()
	}
}

func Use(router *gin.Engine, deps deputil.Dependencies) {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"POST", "GET", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"*"}
	config.ExposeHeaders = []string{"Content-Length"}
	config.AllowCredentials = true
	config.MaxAge = 12 * time.Hour
	router.Use(otelgin.Middleware("autobutler-server"))
	router.Use(cors.New(config))
	router.Use(inject(deps))
	router.Use(trackDevice(deps))
	router.Use(rateLimit())
	router.Use(requireAuth(deps))
}
