package ui

import (
	"autobutler/internal/server/ui/types"
	"autobutler/internal/server/ui/views"
	"autobutler/pkg/storage"

	"github.com/gin-gonic/gin"
)

func SetupDevicesRoutes(router *gin.Engine) {
	setupDevicesView(router)
	setupDevicesComponents(router)
}

func setupDevicesView(router *gin.Engine) {
	uiRoute(router, "/devices", func(c *gin.Context) {
		// Detect storage devices using READ-ONLY operations
		detector := storage.NewDetector()
		devices, err := detector.DetectDevices()
		if err != nil {
			devices = []storage.Device{} // Empty list on error
		}

		// Calculate summary
		summary := detector.CalculateSummary(devices)

		if err := views.Devices(types.NewPageState(), devices, summary).Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
		c.Status(200)
	})
}

func setupDevicesComponents(router *gin.Engine) {
	// Component endpoint for HTMX to refresh just the devices list
	uiRoute(router, "/components/devices/list", func(c *gin.Context) {
		// Re-detect storage devices (READ-ONLY)
		detector := storage.NewDetector()
		devices, err := detector.DetectDevices()
		if err != nil {
			devices = []storage.Device{} // Empty list on error
		}

		// Calculate summary
		summary := detector.CalculateSummary(devices)

		// Render just the devices content component
		if err := views.DevicesContent(devices, summary).Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
		c.Status(200)
	})
}
