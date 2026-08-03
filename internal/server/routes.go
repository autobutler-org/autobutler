package server

import (
	"embed"
	"log/slog"
	"net/http"

	v0_albums "github.com/autobutler-org/autobutler/internal/server/api/v0/albums"
	v0_auth "github.com/autobutler-org/autobutler/internal/server/api/v0/auth"
	v0_books "github.com/autobutler-org/autobutler/internal/server/api/v0/books"
	v0_files "github.com/autobutler-org/autobutler/internal/server/api/v0/cirrus"
	v0_devices "github.com/autobutler-org/autobutler/internal/server/api/v0/devices"
	v0_events "github.com/autobutler-org/autobutler/internal/server/api/v0/events"
	v0_favorites "github.com/autobutler-org/autobutler/internal/server/api/v0/favorites"
	v0_health "github.com/autobutler-org/autobutler/internal/server/api/v0/health"
	v0_metrics "github.com/autobutler-org/autobutler/internal/server/api/v0/metrics"
	v0_migration "github.com/autobutler-org/autobutler/internal/server/api/v0/migration"
	v0_photos "github.com/autobutler-org/autobutler/internal/server/api/v0/photos"
	v0_settings "github.com/autobutler-org/autobutler/internal/server/api/v0/settings"
	v0_shares "github.com/autobutler-org/autobutler/internal/server/api/v0/shares"
	v0_smb "github.com/autobutler-org/autobutler/internal/server/api/v0/smb"
	v0_storage "github.com/autobutler-org/autobutler/internal/server/api/v0/storage"
	v0_thumbnails "github.com/autobutler-org/autobutler/internal/server/api/v0/thumbnails"
	v0_vault "github.com/autobutler-org/autobutler/internal/server/api/v0/vault"
	v0_version "github.com/autobutler-org/autobutler/internal/server/api/v0/version"
	v0_webdav "github.com/autobutler-org/autobutler/internal/server/api/v0/webdav"
	v1_vfs "github.com/autobutler-org/autobutler/internal/server/api/v1/vfs"
	"github.com/autobutler-org/autobutler/pkg/botel/system"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

//go:embed public
var public embed.FS

func setupRoutes(engine *gin.Engine, systemCollector *system.Collector) error {
	setupRouters(engine, systemCollector)
	setupWebDAV(engine)
	return setupStaticRoutes(engine)
}

func setupRouters(engine *gin.Engine, systemCollector *system.Collector) {
	v1group := engine.Group("/api/v1")
	v1Routers := []serverutil.Router{
		v1_vfs.NewRouter(),
	}
	for _, router := range v1Routers {
		serverutil.RegisterRouterWithGroup(v1group, router)
	}

	group := engine.Group("/api/v0")
	apiRouters := []serverutil.Router{
		v0_auth.NewRouter(),
		v0_books.NewRouter(),
		v0_files.NewRouter(),
		v0_devices.NewRouter(),
		v0_events.NewRouter(),
		v0_health.NewRouter(systemCollector),
		v0_metrics.NewRouter(),
		v0_migration.NewRouter(),
		v0_albums.NewRouter(),
		v0_favorites.NewRouter(),
		v0_photos.NewRouter(),
		v0_settings.NewRouter(),
		v0_storage.NewRouter(),
		v0_thumbnails.NewRouter(),
		v0_smb.NewRouter(),
		v0_shares.NewRouter(),
		v0_vault.NewRouter(),
		v0_version.NewRouter(),
	}
	for _, r := range apiRouters {
		serverutil.RegisterRouterWithGroup(group, r)
	}

	// Public share-access route: /s/:token — registered directly on the engine,
	// outside /api/v0, so it bypasses auth middleware (middleware only enforces
	// auth on /api/ and /dav/ prefixes).
	serverutil.RegisterRouter(engine, v0_shares.NewPublicRouter())
}

func setupWebDAV(engine *gin.Engine) {
	cirrusDir, err := storageutil.GetCirrusDir()
	if err != nil {
		// Cirrus dir setup happens earlier in StartServer via setupServices,
		// so this should not fail. Log and skip if it does.
		slog.Error("webdav: failed to get cirrus dir, WebDAV disabled", "err", err)
		return
	}

	handler := v0_webdav.NewHandler(cirrusDir)
	for _, method := range v0_webdav.WebDAVMethods() {
		engine.Handle(method, "/dav/*filepath", handler)
	}
}

func setupStaticRoutes(engine *gin.Engine) error {
	fs, err := static.EmbedFolder(public, "public")
	if err != nil {
		return err
	}
	engine.Use(static.Serve("/", fs))
	// Read index.html once at startup for the SPA fallback.
	// In dev mode (no built frontend), index.html may not exist — fall back
	// to a plain 404 so the server still starts.
	indexHTML, readErr := public.ReadFile("public/index.html")
	if readErr != nil {
		slog.Warn("No embedded index.html — SPA fallback disabled (dev mode?)")
	}
	engine.NoRoute(
		func(c *gin.Context) {
			if indexHTML != nil {
				// Serve index.html for any unmatched route so Flutter's client-side
				// router can read the URL path (e.g. /health, /photos) and navigate.
				// Using c.Data instead of c.FileFromFS to avoid http.FileServer's
				// redirect behavior that strips the path to /.
				c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
				return
			}
			c.String(http.StatusNotFound, "404 page not found")
		},
	)
	return nil
}
