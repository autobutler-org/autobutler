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
		static.Serve("/", staticFS),
		func(c *gin.Context) {
			// TODO: Make this render the SPA index.html instead of a 404 page
			c.FileFromFS("public/index.html", staticFS)
		},
	)
	return nil
}
