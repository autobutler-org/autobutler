package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

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
}
