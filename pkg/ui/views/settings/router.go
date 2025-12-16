package view_settings

import (
	"autobutler/pkg/ui/components/device_manager"
	"autobutler/pkg/ui/components/network_settings"
	"autobutler/pkg/ui/components/user_access"
	"autobutler/pkg/ui/types"

	"autobutler/pkg/util/cirrusutil"
	"autobutler/pkg/util/serverutil"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

type router struct{}

func NewRouter() *router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		settingsRoute,
		settingsUsersRoute,
		settingsNetworkRoute,
		deviceManagerComponentRoute,
		userAccessComponentRoute,
		networkSettingsComponentRoute,
	}
}

var settingsRoute = serverutil.UiRoute(
	"/settings", func(c *gin.Context) templ.Component {
		return Settings(types.NewPageState())
	},
)

var settingsUsersRoute = serverutil.UiRoute(
	"/settings/users", func(c *gin.Context) templ.Component {
		return SettingsUsers(types.NewPageState())
	},
)

var settingsNetworkRoute = serverutil.UiRoute(
	"/settings/network", func(c *gin.Context) templ.Component {
		return SettingsNetwork(types.NewPageState())
	},
)

var deviceManagerComponentRoute = serverutil.UiRoute(
	"/components/settings/device-manager", func(c *gin.Context) templ.Component {
		// Detect all storage devices
		detector := cirrusutil.NewDetector()
		devices, err := detector.DetectDevices()
		if err != nil {
			devices = []cirrusutil.Device{}
		}

		// Get managed devices
		managedDevices, err := cirrusutil.GetManagedDevices()
		if err != nil {
			managedDevices = []cirrusutil.ManagedDevice{}
		}

		return device_manager.Component(devices, managedDevices)
	},
)

var userAccessComponentRoute = serverutil.UiRoute(
	"/components/settings/user-access", func(c *gin.Context) templ.Component {
		// Use mock data for UI display
		users := user_access.GetMockUsers()
		roles := user_access.GetMockRoles()
		permissions := user_access.GetMockPermissions()

		return user_access.Component(users, roles, permissions)
	},
)

var networkSettingsComponentRoute = serverutil.UiRoute(
	"/components/settings/network", func(c *gin.Context) templ.Component {
		// Use mock data for UI display
		status := network_settings.GetMockNetworkStatus()
		devices := network_settings.GetMockTailnetDevices()
		pending := network_settings.GetMockPendingDevices()
		config := network_settings.GetMockNetworkConfig()

		return network_settings.Component(status, devices, pending, config)
	},
)
