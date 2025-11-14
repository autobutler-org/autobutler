package ui

import (
	"autobutler/internal/server/ui/types"
	"autobutler/internal/server/ui/views"
	"autobutler/pkg/storage"

	"github.com/gin-gonic/gin"
)

func SetupIndexRoutes(router *gin.Engine) {
	uiRoute(router, "/", func(c *gin.Context) {
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

		if err := views.Home(types.NewPageState(), summary).Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
		c.Status(200)
	})
}
