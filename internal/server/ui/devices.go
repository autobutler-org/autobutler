package ui

import (
	"autobutler/internal/server/ui/types"
	"autobutler/internal/server/ui/views"
	"autobutler/internal/serverutil"
	"autobutler/pkg/storage"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

func SetupDevicesRoutes(router *gin.Engine) {
	setupDevicesView(router)
	setupDevicesComponents(router)
}

func setupDevicesView(router *gin.Engine) {
	serverutil.UiRoute(router, "/devices", func(c *gin.Context) templ.Component {
		// Detect storage devices using READ-ONLY operations
		detector := storage.NewDetector()
		devices, err := detector.DetectDevices()
		if err != nil {
			devices = []storage.Device{} // Empty list on error
		}

		// Calculate summary
		summary := detector.CalculateSummary(devices)

		return views.Devices(types.NewPageState(), devices, summary)
	})
}

func setupDevicesComponents(router *gin.Engine) {
	// Component endpoint for HTMX to refresh just the devices list
	serverutil.UiRoute(router, "/components/devices/list", func(c *gin.Context) templ.Component {
		// Re-detect storage devices (READ-ONLY)
		detector := storage.NewDetector()
		devices, err := detector.DetectDevices()
		if err != nil {
			devices = []storage.Device{} // Empty list on error
		}
		summary := detector.CalculateSummary(devices)
		return views.DevicesContent(devices, summary)
	})
}
