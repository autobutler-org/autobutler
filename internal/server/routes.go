package server

import (
	"embed"

	v1 "autobutler/internal/server/routes/api/v1"
	"autobutler/internal/server/routes/ui"

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
	v1.SetupFilesRoutes(apiV1Group)
	v1.SetupChatRoutes(apiV1Group)
	v1.SetupUpdateRoutes(apiV1Group)
	v1.SetupHealthRoutes(apiV1Group)
}

func setupStaticRoutes(router *gin.Engine) error {
	staticFS, err := static.EmbedFolder(public, "public")
	if err != nil {
		return err
	}
	router.NoRoute(static.Serve("/public", staticFS))
	return nil
}

func setupUiRoutes(router *gin.Engine) {
	ui.SetupIndexRoutes(router)
	ui.SetupChatRoutes(router)
	ui.SetupFileRoutes(router)
}
