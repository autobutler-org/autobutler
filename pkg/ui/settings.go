package ui

import (
	"autobutler/pkg/ui/components/device_manager"
	"autobutler/pkg/ui/components/network_settings"
	"autobutler/pkg/ui/components/user_access"
	"autobutler/pkg/ui/types"
	"autobutler/pkg/ui/views"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

func SetupSettingsRoutes(router *gin.Engine) {
	setupSettingsView(router)
	setupSettingsUsersView(router)
	setupSettingsNetworkView(router)
	setupThanksView(router)
	setupDeviceManagementComponent(router)
	setupUserAccessComponent(router)
	setupNetworkSettingsComponent(router)
}

func setupSettingsView(router *gin.Engine) {
	serverutil.UiRoute(router, "/settings", func(c *gin.Context) templ.Component {
		return views.Settings(types.NewPageState())
	})
}

func setupSettingsUsersView(router *gin.Engine) {
	serverutil.UiRoute(router, "/settings/users", func(c *gin.Context) templ.Component {
		return views.SettingsUsers(types.NewPageState())
	})
}

func setupSettingsNetworkView(router *gin.Engine) {
	serverutil.UiRoute(router, "/settings/network", func(c *gin.Context) templ.Component {
		return views.SettingsNetwork(types.NewPageState())
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

func setupUserAccessComponent(router *gin.Engine) {
	serverutil.UiRoute(router, "/components/settings/user-access", func(c *gin.Context) templ.Component {
		// Use mock data for UI display
		users := user_access.GetMockUsers()
		roles := user_access.GetMockRoles()
		permissions := user_access.GetMockPermissions()

		return user_access.Component(users, roles, permissions)
	})
}

func setupNetworkSettingsComponent(router *gin.Engine) {
	serverutil.UiRoute(router, "/components/settings/network", func(c *gin.Context) templ.Component {
		// Use mock data for UI display
		status := network_settings.GetMockNetworkStatus()
		devices := network_settings.GetMockTailnetDevices()
		pending := network_settings.GetMockPendingDevices()
		config := network_settings.GetMockNetworkConfig()

		return network_settings.Component(status, devices, pending, config)
	})
}
