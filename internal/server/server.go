package server

import (
	"autobutler/internal/server/middleware"
	"autobutler/pkg/botel"
	"autobutler/pkg/util/deputil"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func StartServer(deps deputil.Dependencies) error {
	tp, err := botel.InitTracer(deps)
	if err != nil {
		return fmt.Errorf("failed to initialize otel trace: %w", err)
	}

	mp, err := botel.InitMetrics(deps)
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

	router := gin.Default()
	// IMPORTANT: middleware.Use MUST be called before setupRoutes
	middleware.Use(router, deps)
	setupRoutes(router)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := router.Run(fmt.Sprintf(":%s", port)); err != nil {
		return err
	}

	return nil
}
