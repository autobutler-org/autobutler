package view_devices

import (
	"autobutler/pkg/ui/types"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

type router struct{}

func NewRouter() *router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		devicesRoute,
		devicesComponentRoute,
	}
}

var devicesRoute = serverutil.UiRoute(
	"/devices", func(c *gin.Context) templ.Component {
		// Detect storage devices using READ-ONLY operations
		detector := storageutil.NewDetector()
		devices, err := detector.DetectDevices()
		if err != nil {
			devices = []storageutil.Device{} // Empty list on error
		}

		// Calculate summary
		summary := storageutil.CalculateSummary(devices)

		return Devices(types.NewPageState(), devices, summary)
	},
)

var devicesComponentRoute = serverutil.UiRoute(
	"/components/devices/list", func(c *gin.Context) templ.Component {
		// Re-detect storage devices (READ-ONLY)
		detector := storageutil.NewDetector()
		devices, err := detector.DetectDevices()
		if err != nil {
			devices = []storageutil.Device{} // Empty list on error
		}
		summary := storageutil.CalculateSummary(devices)
		return DevicesContent(devices, summary)
	},
)
