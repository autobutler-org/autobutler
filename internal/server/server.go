package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	docs "github.com/autobutler-org/autobutler/docs/swagger"
	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/internal/server/middleware"
	"github.com/autobutler-org/autobutler/pkg/backup"
	"github.com/autobutler-org/autobutler/pkg/botel"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/eventbus"
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

	// Build the file index from current filesystem state
	idx := storageutil.NewFileIndex()
	if devices, err := deps.StorageService().GetManagedDevices(); err == nil {
		idx.Build(devices)
	}
	deps.WithFileIndex(idx)

	// Subscribe to events to keep index current
	go func() {
		events, unsub := deps.EventBus().Subscribe("file-index")
		defer unsub()
		for evt := range events {
			devices, err := deps.StorageService().GetManagedDevices()
			if err != nil {
				continue
			}
			var cirrusDir, serial string
			for _, d := range devices {
				s := ""
				if d.UsbInfo != nil {
					s = d.UsbInfo.GetSerial()
				}
				if s == evt.DeviceSerial {
					cirrusDir = d.CirrusDir
					serial = s
					break
				}
			}
			if cirrusDir == "" {
				for _, d := range devices {
					if d.UsbInfo == nil {
						cirrusDir = d.CirrusDir
						serial = ""
						break
					}
				}
			}
			if cirrusDir == "" {
				continue
			}
			switch evt.Kind {
			case eventbus.EventUpload, eventbus.EventNewFolder:
				idx.HandleAdd(cirrusDir, evt.Path, serial)
			case eventbus.EventDelete:
				idx.HandleDelete(cirrusDir, evt.Path)
			case eventbus.EventMove:
				idx.HandleMove(cirrusDir, evt.Path, evt.NewPath, serial)
			}
		}
	}()

	initExternalVault(deps)
	go vaultDeviceMonitor(deps)

	return syncWorker, nil
}

func initExternalVault(deps deputil.Dependencies) {
	serial, err := deps.Database().Queries.GetVaultLocation(context.Background())
	if err != nil || serial == "" {
		return
	}
	device, err := deps.StorageService().FindManagedDeviceBySerial(serial)
	if err != nil || device == nil {
		log.Printf("[vault] external vault device %s not found at startup — vault unavailable until reconnected", serial)
		return
	}
	dbPath := filepath.Join(device.DataDir, "vault.db")
	vaultDB, err := db.ConnectToVaultDatabase(dbPath)
	if err != nil {
		log.Printf("[vault] failed to open external vault db: %v", err)
		return
	}
	deps.SetVaultDB(vaultDB)
	log.Printf("[vault] external vault loaded from device %s", serial)
}

func vaultDeviceMonitor(deps deputil.Dependencies) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	wasConnected := true

	for range ticker.C {
		serial, err := deps.Database().Queries.GetVaultLocation(context.Background())
		if err != nil || serial == "" {
			continue
		}

		device, err := deps.StorageService().FindManagedDeviceBySerial(serial)
		connected := err == nil && device != nil

		if wasConnected && !connected {
			log.Printf("[vault] external device %s disconnected — locking vault", serial)
			deps.VaultSession().LockWithReason("storage device disconnected")
			deps.ClearVaultDB()
			deps.EventBus().Publish(eventbus.Event{
				Kind: eventbus.EventVaultDeviceDisconnected,
				Data: map[string]string{"serial": serial},
			})
			wasConnected = false
		} else if !wasConnected && connected {
			log.Printf("[vault] external device %s reconnected — vault available to unlock", serial)
			dbPath := filepath.Join(device.DataDir, "vault.db")
			vaultDB, err := db.ConnectToVaultDatabase(dbPath)
			if err != nil {
				log.Printf("[vault] failed to reopen vault db: %v", err)
				continue
			}
			deps.SetVaultDB(vaultDB)
			deps.EventBus().Publish(eventbus.Event{
				Kind: eventbus.EventVaultDeviceReconnected,
				Data: map[string]string{"serial": serial},
			})
			wasConnected = true
		}
	}
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
