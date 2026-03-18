package v1_health

import (
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// HealthJSON is the response for the health endpoint.
type HealthJSON struct {
	Healthy            bool     `json:"healthy"`
	Alerts             []string `json:"alerts"`
	CPUPercent         float64  `json:"cpuPercent"`
	MemPercent         float64  `json:"memPercent"`
	DiskPercent        float64  `json:"diskPercent"`
	TemperatureCelsius float64  `json:"temperatureCelsius"`
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
		return serverutil.Ok().WithData(HealthJSON{
			Healthy:            status.Healthy,
			Alerts:             alerts,
			CPUPercent:         status.CPUPercent,
			MemPercent:         status.MemPercent,
			DiskPercent:        status.DiskPercent,
			TemperatureCelsius: status.TemperatureCelsius,
		})
	})
}
