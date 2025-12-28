package server

import (
	"autobutler/internal/server/middleware"
	"autobutler/pkg/botel"
	"autobutler/pkg/networking"
	"autobutler/pkg/util/cirrusutil"
	"autobutler/pkg/util/deputil"
	"autobutler/pkg/util/workerutil"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func setupServices(deps deputil.Dependencies) error {
	if err := cirrusutil.SetupCirrusDir(); err != nil {
		return fmt.Errorf("failed to setup cirrus directory: %w", err)
	}

	node, err := networking.InitNetworkingNode(context.Background())
	if err != nil {
		log.Printf("Warning: failed to initialize networking node: %v", err)
	} else if node != nil {
		deps.WithNetworkingNode(node)
		log.Printf("Networking node initialized successfully")
	}

	go deps.Worker().Process()
	go deps.Worker().LogErrors()
	return nil
}

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

	deps.WithWorker(workerutil.NewWorker())
	if err := setupServices(deps); err != nil {
		return fmt.Errorf("failed to setup services: %w", err)
	}

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
