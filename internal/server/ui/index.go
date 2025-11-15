package ui

import (
	"autobutler/internal/server/ui/types"
	"autobutler/internal/server/ui/views"
	"autobutler/pkg/storage"
	"autobutler/pkg/util/serverutil"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

func SetupIndexRoutes(router *gin.Engine) {
	serverutil.UiRoute(router, "/", func(c *gin.Context) templ.Component {
		// Get storage summary for the storage bar component
		detector := storage.NewDetector()
		devices, err := detector.DetectDevices()
		var summary storage.Summary
		if err == nil && len(devices) > 0 {
			summary = detector.CalculateSummary(devices)
		} else {
			// Provide empty summary if detection fails
			summary = storage.Summary{}
		}

		return views.Home(types.NewPageState(), summary)
	})
}
