package server

import (
	"embed"

	v1_books "autobutler/internal/server/api/v1/books"
	v1_files "autobutler/internal/server/api/v1/cirrus"
	v1_metrics "autobutler/internal/server/api/v1/metrics"
	v1_photos "autobutler/internal/server/api/v1/photos"
	v1_storage "autobutler/internal/server/api/v1/storage"
	v1_thumbnails "autobutler/internal/server/api/v1/thumbnails"
	v1_update "autobutler/internal/server/api/v1/update"
	"autobutler/pkg/ui/types"
	view_books "autobutler/pkg/ui/views/books"
	view_cirrus "autobutler/pkg/ui/views/cirrus"
	view_devices "autobutler/pkg/ui/views/devices"
	view_health "autobutler/pkg/ui/views/health"
	view_home "autobutler/pkg/ui/views/home"
	view_not_found "autobutler/pkg/ui/views/not_found"
	view_photos "autobutler/pkg/ui/views/photos"
	view_settings "autobutler/pkg/ui/views/settings"
	"autobutler/pkg/util/serverutil"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

//go:embed public
var public embed.FS

func setupRoutes(engine *gin.Engine) {
	setupRouters(engine)
	setupStaticRoutes(engine)
}

func setupRouters(engine *gin.Engine) {
	uiRouters := []serverutil.Router{
		view_books.NewRouter(),
		view_devices.NewRouter(),
		view_cirrus.NewRouter(),
		view_health.NewRouter(),
		view_home.NewRouter(),
		view_photos.NewRouter(),
		view_settings.NewRouter(),
	}
	for _, r := range uiRouters {
		serverutil.RegisterRouter(engine, r)
	}

	group := engine.Group("/api/v1")
	apiRouters := []serverutil.Router{
		v1_books.NewRouter(), // Register the new books API router
		v1_files.NewRouter(),
		v1_metrics.NewRouter(),
		v1_storage.NewRouter(),
		v1_thumbnails.NewRouter(),
		v1_update.NewRouter(),
		v1_photos.NewRouter(),
	}
	for _, r := range apiRouters {
		serverutil.RegisterRouterWithGroup(group, r)
	}
}

func setupStaticRoutes(engine *gin.Engine) error {
	staticFS, err := static.EmbedFolder(public, "public")
	if err != nil {
		return err
	}
	engine.NoRoute(
		static.Serve("/public", staticFS),
		func(c *gin.Context) {
			if err := view_not_found.NotFound(types.NewPageState()).Render(c.Request.Context(), c.Writer); err != nil {
				c.Status(400)
				return
			}
			c.Status(404)
		},
	)
	return nil
}
