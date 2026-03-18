package middleware

import (
	"time"

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
}
