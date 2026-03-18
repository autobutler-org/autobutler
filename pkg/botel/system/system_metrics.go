// Package system registers OTel gauges for host hardware metrics using gopsutil.
// Metrics are fed into the existing MeterProvider and stored via the botelsqlite
// exporter, making them queryable through the existing /metrics/query_range API.
package system

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/sensors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	meterName = "autobutler/system"

	// Alert thresholds
	TempCriticalCelsius  = 80.0 // Pi 5 throttle point
	CPUCriticalPercent   = 90.0
	MemCriticalPercent   = 95.0
	DiskCriticalPercent  = 90.0
	CPUSustainedDuration = 60 * time.Second
)

// HealthStatus summarises the current alert state for the health indicator.
type HealthStatus struct {
	Healthy            bool
	Alerts             []string
	CPUPercent         float64
	MemPercent         float64
	MemUsedBytes       uint64
	MemTotalBytes      uint64
	DiskPercent        float64
	DiskUsedBytes      uint64
	DiskTotalBytes     uint64
	TemperatureCelsius float64 // highest thermal zone reading, 0 if unavailable
}

// Collector registers OTel observable gauges and provides HealthStatus polling.
type Collector struct {
	meter        metric.Meter
	cpuHighSince *time.Time
}

// Register creates observable gauges on the global MeterProvider.
// Call this once after InitMetrics().
func Register() (*Collector, error) {
	c := &Collector{
		meter: otel.GetMeterProvider().Meter(meterName),
	}
	if err := c.registerGauges(); err != nil {
		return nil, fmt.Errorf("failed to register system gauges: %w", err)
	}
	return c, nil
}

func (c *Collector) registerGauges() error {
	// CPU utilization per-core + aggregate
	_, err := c.meter.Float64ObservableGauge(
		"system.cpu.utilization",
		metric.WithDescription("CPU utilization [0–100] per core and aggregate"),
		metric.WithUnit("%"),
		metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
			percents, err := cpu.Percent(0, true)
			if err != nil {
				slog.Warn("system metrics: cpu.Percent failed", "err", err)
				return nil
			}
			for i, p := range percents {
				o.Observe(p, metric.WithAttributes(
					attribute.String("core", fmt.Sprintf("cpu%d", i)),
				))
			}
			// Aggregate
			agg, err := cpu.Percent(0, false)
			if err == nil && len(agg) > 0 {
				o.Observe(agg[0], metric.WithAttributes(
					attribute.String("core", "aggregate"),
				))
			}
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("cpu gauge: %w", err)
	}

	// Memory utilization
	_, err = c.meter.Float64ObservableGauge(
		"system.memory.utilization",
		metric.WithDescription("Memory utilization [0–100]"),
		metric.WithUnit("%"),
		metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
			v, err := mem.VirtualMemory()
			if err != nil {
				slog.Warn("system metrics: mem.VirtualMemory failed", "err", err)
				return nil
			}
			o.Observe(v.UsedPercent)
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("memory gauge: %w", err)
	}

	// Disk utilization (root mount + any additional mounts)
	_, err = c.meter.Float64ObservableGauge(
		"system.disk.utilization",
		metric.WithDescription("Disk utilization [0–100] per mount point"),
		metric.WithUnit("%"),
		metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
			partitions, err := disk.Partitions(false)
			if err != nil {
				slog.Warn("system metrics: disk.Partitions failed", "err", err)
				return nil
			}
			for _, p := range partitions {
				usage, err := disk.Usage(p.Mountpoint)
				if err != nil {
					continue
				}
				o.Observe(usage.UsedPercent, metric.WithAttributes(
					attribute.String("mount", p.Mountpoint),
				))
			}
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("disk gauge: %w", err)
	}

	// Temperature (thermal zones — available on Pi via /sys/class/thermal)
	_, err = c.meter.Float64ObservableGauge(
		"system.temperature",
		metric.WithDescription("Hardware temperature in Celsius per thermal zone"),
		metric.WithUnit("Cel"),
		metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
			temps, err := sensors.SensorsTemperatures()
			if err != nil {
				// Not fatal — not available in all environments (e.g. Docker without /sys)
				return nil
			}
			for _, t := range temps {
				if t.Temperature <= 0 {
					continue
				}
				o.Observe(t.Temperature, metric.WithAttributes(
					attribute.String("sensor", t.SensorKey),
				))
			}
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("temperature gauge: %w", err)
	}

	// Load averages
	_, err = c.meter.Float64ObservableGauge(
		"system.load.average",
		metric.WithDescription("System load average"),
		metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
			avg, err := load.Avg()
			if err != nil {
				slog.Warn("system metrics: load.Avg failed", "err", err)
				return nil
			}
			o.Observe(avg.Load1, metric.WithAttributes(attribute.String("interval", "1m")))
			o.Observe(avg.Load5, metric.WithAttributes(attribute.String("interval", "5m")))
			o.Observe(avg.Load15, metric.WithAttributes(attribute.String("interval", "15m")))
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("load gauge: %w", err)
	}

	return nil
}

// CurrentHealth samples system state directly (no DB needed) for the health endpoint.
func (c *Collector) CurrentHealth() HealthStatus {
	status := HealthStatus{Healthy: true}

	// CPU
	if agg, err := cpu.Percent(100*time.Millisecond, false); err == nil && len(agg) > 0 {
		status.CPUPercent = agg[0]
		if agg[0] >= CPUCriticalPercent {
			now := time.Now()
			if c.cpuHighSince == nil {
				c.cpuHighSince = &now
			} else if time.Since(*c.cpuHighSince) >= CPUSustainedDuration {
				status.Healthy = false
				status.Alerts = append(status.Alerts,
					fmt.Sprintf("CPU sustained above %.0f%% for >%s", CPUCriticalPercent, CPUSustainedDuration))
			}
		} else {
			c.cpuHighSince = nil
		}
	}

	// Memory
	if v, err := mem.VirtualMemory(); err == nil {
		status.MemPercent = v.UsedPercent
		status.MemUsedBytes = v.Used
		status.MemTotalBytes = v.Total
		if v.UsedPercent >= MemCriticalPercent {
			status.Healthy = false
			status.Alerts = append(status.Alerts,
				fmt.Sprintf("Memory usage critical: %.1f%%", v.UsedPercent))
		}
	}

	// Disk (root)
	if usage, err := disk.Usage("/"); err == nil {
		status.DiskPercent = usage.UsedPercent
		status.DiskUsedBytes = usage.Used
		status.DiskTotalBytes = usage.Total
		if usage.UsedPercent >= DiskCriticalPercent {
			status.Healthy = false
			status.Alerts = append(status.Alerts,
				fmt.Sprintf("Disk usage critical: %.1f%%", usage.UsedPercent))
		}
	}

	// Temperature (highest reading)
	if temps, err := sensors.SensorsTemperatures(); err == nil {
		var maxTemp float64
		for _, t := range temps {
			if t.Temperature > maxTemp {
				maxTemp = t.Temperature
			}
		}
		status.TemperatureCelsius = maxTemp
		if maxTemp >= TempCriticalCelsius {
			status.Healthy = false
			status.Alerts = append(status.Alerts,
				fmt.Sprintf("Temperature critical: %.1f°C", maxTemp))
		}
	}

	return status
}
