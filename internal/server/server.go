package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	docs "github.com/autobutler-org/autobutler/docs/swagger"
	"github.com/autobutler-org/autobutler/internal/server/middleware"
	"github.com/autobutler-org/autobutler/pkg/backup"
	"github.com/autobutler-org/autobutler/pkg/botel"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/favoritesutil"
	"github.com/autobutler-org/autobutler/pkg/util/remoteutil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/settingsutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
	"github.com/autobutler-org/autobutler/pkg/util/workerutil"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func setupServices(deps deputil.Dependencies) (*backup.SyncWorker, error) {
	if err := storageutil.SetupCirrusDir(); err != nil {
		return nil, fmt.Errorf("failed to setup cirrus directory: %w", err)
	}
	go deps.Worker().Process()
	go deps.Worker().LogErrors()
	if _, err := favoritesutil.EnsureFavoritesAlbum(
		context.Background(),
		deps.Database().Queries,
	); err != nil {
		log.Printf("[server] warning: could not ensure Favorites album: %v", err)
	}

	syncWorker := backup.NewSyncWorker(backup.SyncWorkerParams{
		Bus:     deps.EventBus(),
		Storage: deps.StorageService(),
		Queries: deps.Database().Queries,
	})
	syncWorker.Start()

	return syncWorker, nil
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
	syncWorker, err := setupServices(deps)
	if err != nil {
		return fmt.Errorf("failed to setup services: %w", err)
	}

	portNum := serverutil.ServerPort()
	port := fmt.Sprintf("%d", portNum)

	if enabled, authKey := settingsutil.GetRemoteAccess(); enabled && authKey != "" {
		if err := remoteutil.Start(authKey); err != nil {
			log.Printf("[remote] failed to start: %v", err)
		} else if err := remoteutil.StartProxy(portNum); err != nil {
			log.Printf("[remote] failed to start proxy: %v", err)
		}
	}

	// Graceful shutdown: stop tsnet and telemetry on SIGINT/SIGTERM.
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("[server] shutting down...")
		syncWorker.Stop()
		remoteutil.Stop()
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
		if err := mp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down meter provider: %v", err)
		}
		os.Exit(0)
	}()

	router := gin.Default()
	// Disable automatic redirects so unmatched routes (e.g. /health, /photos)
	// fall through to the NoRoute SPA handler instead of 301-redirecting to /.
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	// IMPORTANT: middleware.Use MUST be called before setupRoutes
	middleware.Use(router, deps)
	setupRoutes(router, systemCollector)
	setupSwagger(router)

	if err := router.Run(fmt.Sprintf(":%s", port)); err != nil {
		return err
	}

	return nil
}
