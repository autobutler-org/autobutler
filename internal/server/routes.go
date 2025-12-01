package server

import (
	"embed"

	v1_files "autobutler/internal/server/api/v1/files"
	v1_metrics "autobutler/internal/server/api/v1/metrics"
	v1_storage "autobutler/internal/server/api/v1/storage"
	v1_thumbnails "autobutler/internal/server/api/v1/thumbnails"
	v1_update "autobutler/internal/server/api/v1/update"
	"autobutler/pkg/ui"
	"autobutler/pkg/ui/types"
	"autobutler/pkg/ui/views"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

//go:embed public
var public embed.FS

func setupRoutes(router *gin.Engine) {
	setupApiRoutes(router)
	setupStaticRoutes(router)
	setupUiRoutes(router)
}

func setupApiRoutes(router *gin.Engine) {
	group := router.Group("/api/v1")
	v1_files.SetupRoutes(group)
	v1_metrics.SetupRoutes(group)
	v1_storage.SetupRoutes(group)
	v1_thumbnails.SetupRoutes(group)
	v1_update.SetupRoutes(group)
}

func setupStaticRoutes(router *gin.Engine) error {
	staticFS, err := static.EmbedFolder(public, "public")
	if err != nil {
		return err
	}
	router.NoRoute(
		static.Serve("/public", staticFS),
		func(c *gin.Context) {
			if err := views.NotFound(types.NewPageState()).Render(c.Request.Context(), c.Writer); err != nil {
				c.Status(400)
				return
			}
			c.Status(404)
		},
	)
	return nil
}

func setupUiRoutes(router *gin.Engine) {
	ui.SetupBookRoutes(router)
	ui.SetupDevicesRoutes(router)
	ui.SetupFileRoutes(router)
	ui.SetupHealthRoutes(router)
	ui.SetupIndexRoutes(router)
	ui.SetupPhotoRoutes(router)
	ui.SetupSettingsRoutes(router)
}
