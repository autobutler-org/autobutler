package middleware

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/util/authutil"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

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

// requireAuth validates the session token from cookie or Authorization header.
// Exempt paths (setup, login, recover, status) are always allowed through.
// If no users have been set up yet, all requests are allowed through (first-boot).
func requireAuth(deps deputil.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if authExemptPaths[path] {
			c.Next()
			return
		}

		db := deps.Database()
		if db == nil {
			c.Next()
			return
		}

		// Allow all requests through if setup hasn't been completed yet
		complete, err := authutil.IsSetupComplete(c.Request.Context(), db.Queries)
		if err != nil || !complete {
			c.Next()
			return
		}

		// Extract token from Authorization header or cookie
		token := ""
		if auth := c.GetHeader("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimPrefix(auth, "Bearer ")
		} else if cookie, err := c.Cookie("session"); err == nil {
			token = cookie
		}

		if token == "" {
			c.JSON(401, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}

		username, err := authutil.ValidateSession(c.Request.Context(), db.Queries, token)
		if err != nil {
			c.JSON(401, gin.H{"error": "invalid or expired session"})
			c.Abort()
			return
		}

		// Inject username into context for downstream handlers
		c = ctxutil.With(c, "username", username)
		c.Next()
	}
}

func Use(router *gin.Engine, deps deputil.Dependencies) {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"POST", "GET", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"*"}
	config.ExposeHeaders = []string{"Content-Length"}
	config.AllowCredentials = true
	config.MaxAge = 12 * time.Hour
	router.Use(otelgin.Middleware("autobutler-server"))
	router.Use(cors.New(config))
	router.Use(inject(deps))
	router.Use(trackDevice(deps))
	router.Use(requireAuth(deps))
}
