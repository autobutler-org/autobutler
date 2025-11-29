package ui

import (
	"autobutler/pkg/util/storageutil"
	"autobutler/pkg/ui/components/device_manager"
	"autobutler/pkg/ui/types"
	"autobutler/pkg/ui/views"
	"autobutler/pkg/util/serverutil"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

func SetupSettingsRoutes(router *gin.Engine) {
	setupSettingsView(router)
	setupThanksView(router)
	setupDeviceManagementComponent(router)
}

func setupSettingsView(router *gin.Engine) {
	serverutil.UiRoute(router, "/settings", func(c *gin.Context) templ.Component {
		return views.Settings(types.NewPageState())
	})
}

func setupThanksView(router *gin.Engine) {
	serverutil.UiRoute(router, "/thanks", func(c *gin.Context) templ.Component {
		return views.Thanks(types.NewPageState())
	})
}

func setupDeviceManagementComponent(router *gin.Engine) {
	serverutil.UiRoute(router, "/components/settings/device-manager", func(c *gin.Context) templ.Component {
		// Detect all storage devices
		detector := storageutil.NewDetector()
		devices, err := detector.DetectDevices()
		if err != nil {
			devices = []storageutil.Device{}
		}

		// Get managed devices
		managedDevices, err := storageutil.GetManagedDevices()
		if err != nil {
			managedDevices = []storageutil.ManagedDevice{}
		}

		return device_manager.Component(devices, managedDevices)
	})
}
