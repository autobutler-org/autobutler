package v0_health

import (
	"os"

	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/smartutil"
	"github.com/gin-gonic/gin"
)

// HealthJSON is the response for the health endpoint.
type HealthJSON struct {
	Healthy            bool      `json:"healthy"`
	Alerts             []string  `json:"alerts"`
	CPUPercent         float64   `json:"cpuPercent"`
	CPUCorePercents    []float64 `json:"cpuCorePercents"`
	CPUCoreCount       int       `json:"cpuCoreCount"`
	MemPercent         float64   `json:"memPercent"`
	MemUsedBytes       uint64    `json:"memUsedBytes"`
	MemTotalBytes      uint64    `json:"memTotalBytes"`
	DiskPercent        float64   `json:"diskPercent"`
	DiskUsedBytes      uint64    `json:"diskUsedBytes"`
	DiskTotalBytes     uint64    `json:"diskTotalBytes"`
	TemperatureCelsius float64   `json:"temperatureCelsius"`
	// SMARTDrives contains per-drive S.M.A.R.T. health data. Empty when
	// smartctl is unavailable or no devices are configured.
	SMARTDrives []smartutil.DriveHealth `json:"smartDrives,omitempty"`
	// Hostname is the OS hostname of the butler device.
	// Clients can use this to display accurate LAN mount paths (e.g. smb://hostname.local).
	Hostname string `json:"hostname"`
}

// getHealth godoc
// @Summary Get system health status
// @Description Returns current hardware health: CPU, memory, disk usage and temperature. Sets healthy=false with alert messages when any metric exceeds its critical threshold.
// @Tags health
// @Produce json
// @Success 200 {object} HealthJSON
// @Router /health [get]
func (r *router) getHealthRoute() *serverutil.Route {
	return serverutil.ApiRoute("GET", "/health", func(c *gin.Context) *serverutil.Response {
		status := r.collector.CurrentHealth()
		alerts := status.Alerts
		if alerts == nil {
			alerts = []string{}
		}
		corePercents := status.CPUCorePercents
		if corePercents == nil {
			corePercents = []float64{}
		}
		// omitempty on a nil slice would drop the key entirely; normalise to an
		// empty slice so clients always see a consistent shape.
		smartDrives := status.SMARTDrives
		if smartDrives == nil {
			smartDrives = []smartutil.DriveHealth{}
		}
		hostname, _ := os.Hostname()
		return serverutil.Ok().WithData(HealthJSON{
			Healthy:            status.Healthy,
			Alerts:             alerts,
			CPUPercent:         status.CPUPercent,
			CPUCorePercents:    corePercents,
			CPUCoreCount:       len(corePercents),
			MemPercent:         status.MemPercent,
			MemUsedBytes:       status.MemUsedBytes,
			MemTotalBytes:      status.MemTotalBytes,
			DiskPercent:        status.DiskPercent,
			DiskUsedBytes:      status.DiskUsedBytes,
			DiskTotalBytes:     status.DiskTotalBytes,
			TemperatureCelsius: status.TemperatureCelsius,
			SMARTDrives:        smartDrives,
			Hostname:           hostname,
		})
	})
}
