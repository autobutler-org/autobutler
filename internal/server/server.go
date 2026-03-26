package server

import (
	"context"
	"fmt"
	"log"
	"os"

	docs "github.com/autobutler-org/autobutler/docs/swagger"
	"github.com/autobutler-org/autobutler/internal/server/middleware"
	"github.com/autobutler-org/autobutler/pkg/botel"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
	"github.com/autobutler-org/autobutler/pkg/util/workerutil"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func setupServices(deps deputil.Dependencies) error {
	if err := storageutil.SetupCirrusDir(); err != nil {
		return fmt.Errorf("failed to setup cirrus directory: %w", err)
	}
	go deps.Worker().Process()
	go deps.Worker().LogErrors()
	return nil
}

func setupSwagger(router *gin.Engine) {
	docs.SwaggerInfo.BasePath = "/api/v1"
	router.GET("/swagger", func(c *gin.Context) {
		c.Redirect(302, "/swagger/index.html")
	})
	router.GET("/swagger/", func(c *gin.Context) {
		c.Redirect(302, "/swagger/index.html")
	})
	router.GET("/swagger/:any", ginSwagger.WrapHandler(swaggerfiles.Handler))
}

func StartServer(deps deputil.Dependencies) error {
	tp, err := botel.InitTracer(deps)
	if err != nil {
		return fmt.Errorf("failed to initialize otel trace: %w", err)
	}

	mp, systemCollector, err := botel.InitMetrics(deps)
	if err != nil {
		return fmt.Errorf("failed to initialize otel metrics: %w", err)
	}

	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
		if err := mp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down meter provider: %v", err)
		}
	}()

	deps.WithWorker(workerutil.NewWorker(deps.StorageService()))
	if err := setupServices(deps); err != nil {
		return fmt.Errorf("failed to setup services: %w", err)
	}

	router := gin.Default()
	// Disable automatic redirects so unmatched routes (e.g. /health, /photos)
	// fall through to the NoRoute SPA handler instead of 301-redirecting to /.
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	// IMPORTANT: middleware.Use MUST be called before setupRoutes
	middleware.Use(router, deps)
	setupRoutes(router, systemCollector)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	setupSwagger(router)

	if err := router.Run(fmt.Sprintf(":%s", port)); err != nil {
		return err
	}

	return nil
}
