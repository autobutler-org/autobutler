package middleware

import (
	"autobutler/pkg/util/ctxutil"
	"autobutler/pkg/util/deputil"
	"time"

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

// redirectLegacyPaths redirects old /files paths to /cirrus for backward compatibility
func redirectLegacyPaths() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Redirect /files to /cirrus
		if path == "/files" || path == "/files/" {
			c.Redirect(301, "/cirrus")
			c.Abort()
			return
		}

		// Redirect /files/* to /cirrus/*
		if len(path) > 6 && path[:7] == "/files/" {
			newPath := "/cirrus" + path[6:]
			c.Redirect(301, newPath)
			c.Abort()
			return
		}

		// Redirect /api/v1/files to /api/v1/cirrus
		if path == "/api/v1/files" || path == "/api/v1/files/" {
			c.Redirect(301, "/api/v1/cirrus")
			c.Abort()
			return
		}

		// Redirect /api/v1/files/* to /api/v1/cirrus/*
		if len(path) > 14 && path[:15] == "/api/v1/files/" {
			newPath := "/api/v1/cirrus" + path[14:]
			c.Redirect(301, newPath)
			c.Abort()
			return
		}

		// Redirect /components/files/* to /components/cirrus/*
		if len(path) > 16 && path[:17] == "/components/files" {
			newPath := "/components/cirrus" + path[17:]
			c.Redirect(301, newPath)
			c.Abort()
			return
		}

		// Redirect /api/v1/folder/files/* to /api/v1/folder/cirrus/*
		if len(path) > 21 && path[:22] == "/api/v1/folder/files/" {
			newPath := "/api/v1/folder/cirrus" + path[21:]
			c.Redirect(301, newPath)
			c.Abort()
			return
		}

		c.Next()
	}
}

func Use(router *gin.Engine, deps deputil.Dependencies) {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"POST", "GET", "PUT", "OPTIONS"}
	config.AllowHeaders = []string{"*"}
	config.ExposeHeaders = []string{"Content-Length"}
	config.AllowCredentials = true
	config.MaxAge = 12 * time.Hour
	router.Use(otelgin.Middleware("autobutler-server"))
	router.Use(cors.New(config))

	router.Use(redirectLegacyPaths())
	router.Use(inject(deps))
}
