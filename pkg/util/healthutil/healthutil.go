// Package healthutil provides gopsutil-based host hardware health metrics.
// It is intentionally free of OTel dependencies so the health endpoint
// has no dependency on any telemetry backend.
package healthutil

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/sensors"
)

const (
	// Alert thresholds
	TempCriticalCelsius  = 80.0 // Pi 5 throttle point
	CPUCriticalPercent   = 90.0
	MemCriticalPercent   = 95.0
	DiskCriticalPercent  = 90.0
	CPUSustainedDuration = 60 * time.Second
)

// HealthStatus summarizes the current alert state for the health indicator.
type HealthStatus struct {
	Healthy            bool
	Alerts             []string
	CPUPercent         float64
	CPUCorePercents    []float64 // per-core utilization
	MemPercent         float64
	MemUsedBytes       uint64
	MemTotalBytes      uint64
	DiskPercent        float64
	DiskUsedBytes      uint64
	DiskTotalBytes     uint64
	TemperatureCelsius float64 // highest thermal zone reading, 0 if unavailable
}

// Collector samples host hardware metrics for the health endpoint.
type Collector struct {
	cpuHighSince *time.Time
}

// Register creates a new Collector. Call this once at startup.
func Register() (*Collector, error) {
	return &Collector{}, nil
}

// The applyXThreshold helpers hold the alerting rules, split out from the
// gopsutil sampling in CurrentHealth so they can be tested without a machine
// that is genuinely at 95% memory or 80°C. Each mutates status in place,
// clearing Healthy and appending an alert when its limit is breached.

// applyCPUThreshold implements the sustained-load rule: CPU must stay at or
// above CPUCriticalPercent for longer than CPUSustainedDuration before it
// alerts, so a brief spike does not mark the host unhealthy. Returns the
// updated "high since" marker — nil once CPU drops back below the limit.
func applyCPUThreshold(status *HealthStatus, cpuPercent float64, highSince *time.Time, now time.Time) *time.Time {
	if cpuPercent < CPUCriticalPercent {
		return nil
	}
	if highSince == nil {
		return &now
	}
	if now.Sub(*highSince) >= CPUSustainedDuration {
		status.Healthy = false
		status.Alerts = append(status.Alerts,
			fmt.Sprintf("CPU sustained above %.0f%% for >%s", CPUCriticalPercent, CPUSustainedDuration))
	}
	return highSince
}

// applyMemThreshold alerts when memory use reaches MemCriticalPercent.
func applyMemThreshold(status *HealthStatus, memPercent float64) {
	if memPercent >= MemCriticalPercent {
		status.Healthy = false
		status.Alerts = append(status.Alerts,
			fmt.Sprintf("Memory usage critical: %.1f%%", memPercent))
	}
}

// applyDiskThreshold alerts when root filesystem use reaches DiskCriticalPercent.
func applyDiskThreshold(status *HealthStatus, diskPercent float64) {
	if diskPercent >= DiskCriticalPercent {
		status.Healthy = false
		status.Alerts = append(status.Alerts,
			fmt.Sprintf("Disk usage critical: %.1f%%", diskPercent))
	}
}

// applyTempThreshold alerts when the hottest thermal zone reaches
// TempCriticalCelsius (the Pi 5 throttle point).
func applyTempThreshold(status *HealthStatus, tempCelsius float64) {
	if tempCelsius >= TempCriticalCelsius {
		status.Healthy = false
		status.Alerts = append(status.Alerts,
			fmt.Sprintf("Temperature critical: %.1f°C", tempCelsius))
	}
}

// CurrentHealth samples system state directly via gopsutil for the health endpoint.
func (c *Collector) CurrentHealth() HealthStatus {
	status := HealthStatus{Healthy: true}

	// CPU
	if cores, err := cpu.Percent(100*time.Millisecond, true); err == nil {
		status.CPUCorePercents = cores
	} else {
		slog.Warn("system metrics: cpu.Percent (per-core) failed", "err", err)
	}
	if agg, err := cpu.Percent(0, false); err == nil && len(agg) > 0 {
		status.CPUPercent = agg[0]
		c.cpuHighSince = applyCPUThreshold(&status, agg[0], c.cpuHighSince, time.Now())
	} else {
		slog.Warn("system metrics: cpu.Percent (aggregate) failed", "err", err)
	}

	// Memory
	if v, err := mem.VirtualMemory(); err == nil {
		status.MemPercent = v.UsedPercent
		status.MemUsedBytes = v.Used
		status.MemTotalBytes = v.Total
		applyMemThreshold(&status, v.UsedPercent)
	} else {
		slog.Warn("system metrics: mem.VirtualMemory failed", "err", err)
	}

	// Disk (root)
	if usage, err := disk.Usage("/"); err == nil {
		status.DiskPercent = usage.UsedPercent
		status.DiskUsedBytes = usage.Used
		status.DiskTotalBytes = usage.Total
		applyDiskThreshold(&status, usage.UsedPercent)
	} else {
		slog.Warn("system metrics: disk.Usage failed", "err", err)
	}

	// Temperature (highest reading across all thermal zones)
	if temps, err := sensors.SensorsTemperatures(); err == nil {
		var maxTemp float64
		for _, t := range temps {
			if t.Temperature > maxTemp {
				maxTemp = t.Temperature
			}
		}
		status.TemperatureCelsius = maxTemp
		applyTempThreshold(&status, maxTemp)
	}
	// Temperature failure is non-fatal: not available in all environments.

	// Load averages — informational only, no alert threshold
	if avg, err := load.Avg(); err != nil {
		slog.Warn("system metrics: load.Avg failed", "err", err)
	} else {
		_ = avg // available if callers want to extend HealthStatus later
	}

	return status
}
