package server

import (
	"embed"
	"log/slog"
	"net/http"

	v0_admin "github.com/autobutler-org/autobutler/internal/server/api/v0/admin"
	v0_albums "github.com/autobutler-org/autobutler/internal/server/api/v0/albums"
	v0_auth "github.com/autobutler-org/autobutler/internal/server/api/v0/auth"
	v0_books "github.com/autobutler-org/autobutler/internal/server/api/v0/books"
	v0_files "github.com/autobutler-org/autobutler/internal/server/api/v0/cirrus"
	v0_devices "github.com/autobutler-org/autobutler/internal/server/api/v0/devices"
	v0_events "github.com/autobutler-org/autobutler/internal/server/api/v0/events"
	v0_favorites "github.com/autobutler-org/autobutler/internal/server/api/v0/favorites"
	v0_health "github.com/autobutler-org/autobutler/internal/server/api/v0/health"

	v0_migration "github.com/autobutler-org/autobutler/internal/server/api/v0/migration"
	v0_photos "github.com/autobutler-org/autobutler/internal/server/api/v0/photos"
	v0_settings "github.com/autobutler-org/autobutler/internal/server/api/v0/settings"
	v0_smb "github.com/autobutler-org/autobutler/internal/server/api/v0/smb"
	v0_storage "github.com/autobutler-org/autobutler/internal/server/api/v0/storage"
	v0_thumbnails "github.com/autobutler-org/autobutler/internal/server/api/v0/thumbnails"
	v0_vault "github.com/autobutler-org/autobutler/internal/server/api/v0/vault"
	v0_version "github.com/autobutler-org/autobutler/internal/server/api/v0/version"
	v0_webdav "github.com/autobutler-org/autobutler/internal/server/api/v0/webdav"
	v1_vfs "github.com/autobutler-org/autobutler/internal/server/api/v1/vfs"
	"github.com/autobutler-org/autobutler/internal/server/middleware"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/healthutil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

//go:embed public
var public embed.FS

func setupRoutes(engine *gin.Engine, systemCollector *healthutil.Collector, deps deputil.Dependencies) error {
	setupRouters(engine, systemCollector, deps)
	setupWebDAV(engine)
	return setupStaticRoutes(engine)
}

func setupRouters(engine *gin.Engine, systemCollector *healthutil.Collector, deps deputil.Dependencies) {
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
		v0_migration.NewRouter(),
		v0_albums.NewRouter(),
		v0_favorites.NewRouter(),
		v0_photos.NewRouter(),
		v0_settings.NewRouter(),
		v0_storage.NewRouter(),
		v0_thumbnails.NewRouter(),
		v0_smb.NewRouter(),
		v0_vault.NewRouter(),
		v0_version.NewRouter(),
	}
	for _, r := range apiRouters {
		serverutil.RegisterRouterWithGroup(group, r)
	}

	// Admin-only routes — wrapped with RequireAdmin middleware.
	adminGroup := group.Group("", middleware.RequireAdmin(deps))
	serverutil.RegisterRouterWithGroup(adminGroup, v0_admin.NewRouter())
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
	// /mobile is a minimal pairing page. Registered before the static handler
	// so it takes precedence over the Flutter SPA's catch-all.
	engine.GET("/mobile", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(mobilePage))
	})
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

// mobilePage is the HTML served at /mobile — a minimal page that requests a
// QR pairing code from the API and renders it for scanning by a new device.
// Requires an authenticated session (the admin loads this page from their
// already-connected browser to onboard a phone).
const mobilePage = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Pair a device — AutoButler</title>
<style>
  body { font-family: system-ui, sans-serif; max-width: 480px; margin: 3rem auto; text-align: center; color: #1a1a2e; }
  h1 { font-size: 1.4rem; margin-bottom: .5rem; }
  p  { color: #555; font-size: .95rem; margin: .4rem 0 1.5rem; }
  img { border: 2px solid #e0e0e0; border-radius: 8px; max-width: 100%; }
  button { margin-top: 1.5rem; padding: .6rem 1.4rem; border-radius: 6px;
           border: none; background: #1a73e8; color: #fff; cursor: pointer;
           font-size: 1rem; }
  button:hover { background: #1557b0; }
  .msg { margin-top: 1rem; font-size: .9rem; color: #888; }
</style>
</head>
<body>
<h1>🦎 Pair a new device</h1>
<p>Scan the QR code with your phone's camera app to connect it to AutoButler.<br>
The code expires in <strong>10 minutes</strong> and can only be used once.</p>
<div id="qr"><p class="msg">Loading…</p></div>
<button onclick="refresh()">Generate new code</button>
<p class="msg" id="status"></p>
<script>
async function refresh() {
  document.getElementById('qr').innerHTML = '<p class="msg">Loading…</p>';
  document.getElementById('status').textContent = '';
  try {
    const resp = await fetch('/api/v0/auth/pairing');
    if (resp.status === 401) {
      document.getElementById('qr').innerHTML = '<p class="msg">Not authenticated. <a href="/">Log in</a> first.</p>';
      return;
    }
    if (!resp.ok) throw new Error('Server error: ' + resp.status);
    const blob = await resp.blob();
    const url = URL.createObjectURL(blob);
    document.getElementById('qr').innerHTML = '<img src="' + url + '" alt="QR pairing code" width="256" height="256">';
    document.getElementById('status').textContent = 'Generated at ' + new Date().toLocaleTimeString();
  } catch(e) {
    document.getElementById('qr').innerHTML = '<p class="msg">Error: ' + e.message + '</p>';
  }
}
refresh();
// Auto-refresh every 8 minutes to stay ahead of the 10-min expiry.
setInterval(refresh, 8 * 60 * 1000);
</script>
</body>
</html>`
