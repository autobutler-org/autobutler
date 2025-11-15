package server

import (
	"embed"

	v1 "autobutler/internal/server/api/v1"
	"autobutler/internal/server/ui"
	"autobutler/internal/server/ui/types"
	"autobutler/internal/server/ui/views"

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
	apiV1Group := router.Group("/api/v1")
	v1.SetupMetricsRoutes(apiV1Group, metricsExporter)
	v1.SetupDocRoutes(apiV1Group)
	v1.SetupFilesRoutes(apiV1Group)
	v1.SetupCalendarRoutes(apiV1Group)
	v1.SetupStorageRoutes(apiV1Group)
	v1.SetupUpdateRoutes(apiV1Group)
	v1.SetupHealthRoutes(apiV1Group)
	v1.SetupThumbnailRoutes(apiV1Group)
}

func setupStaticRoutes(router *gin.Engine) error {
	staticFS, err := static.EmbedFolder(public, "public")
	if err != nil {
		return err
	}
	router.NoRoute(
		static.Serve("/public", staticFS),
		// TODO: have a proper 404 page
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
	ui.SetupHealthRoutes(router)
	ui.SetupIndexRoutes(router)
	ui.SetupCalendarRoutes(router)
	ui.SetupDevicesRoutes(router)
	ui.SetupFileRoutes(router)
	ui.SetupPhotoRoutes(router)
	ui.SetupBookRoutes(router)
}
