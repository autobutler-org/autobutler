package healthutil

import (
	"fmt"
	"time"
)

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
