package ui

import (
	"autobutler/pkg/ui/types"
	"autobutler/pkg/ui/views"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

func SetupIndexRoutes(router *gin.Engine) {
	serverutil.UiRoute(router, "/", func(c *gin.Context) templ.Component {
		// Get storage summary for the storage bar component
		detector := storageutil.NewDetector()
		devices, err := detector.DetectDevices()
		var summary storageutil.Summary
		if err == nil && len(devices) > 0 {
			summary = storageutil.CalculateSummary(devices)
		} else {
			// Provide empty summary if detection fails
			summary = storageutil.Summary{}
		}

		return views.Home(types.NewPageState(), summary)
	})
}
