package ui

import (
	"autobutler/pkg/util/storageutil"
	"autobutler/pkg/ui/types"
	"autobutler/pkg/ui/views"
	"autobutler/pkg/util/serverutil"

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
		detector := storageutil.NewDetector()
		devices, err := detector.DetectDevices()
		if err != nil {
			devices = []storageutil.Device{} // Empty list on error
		}

		// Calculate summary
		summary := storageutil.CalculateSummary(devices)

		return views.Devices(types.NewPageState(), devices, summary)
	})
}

func setupDevicesComponents(router *gin.Engine) {
	// Component endpoint for HTMX to refresh just the devices list
	serverutil.UiRoute(router, "/components/devices/list", func(c *gin.Context) templ.Component {
		// Re-detect storage devices (READ-ONLY)
		detector := storageutil.NewDetector()
		devices, err := detector.DetectDevices()
		if err != nil {
			devices = []storageutil.Device{} // Empty list on error
		}
		summary := storageutil.CalculateSummary(devices)
		return views.DevicesContent(devices, summary)
	})
}
