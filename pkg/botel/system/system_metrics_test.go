package system

import (
	"strings"
	"testing"
)

// TestApplyThresholds_Healthy verifies that a status with all values below
// critical thresholds remains healthy with no alerts.
func TestApplyThresholds_Healthy(t *testing.T) {
	s := HealthStatus{
		Healthy:            true,
		MemPercent:         50.0,
		DiskPercent:        60.0,
		TemperatureCelsius: 55.0,
	}
	applyThresholds(&s)

	if !s.Healthy {
		t.Error("expected Healthy=true for below-threshold values")
	}
	if len(s.Alerts) != 0 {
		t.Errorf("expected no alerts, got: %v", s.Alerts)
	}
}

// TestApplyThresholds_MemCritical verifies the memory threshold fires correctly.
func TestApplyThresholds_MemCritical(t *testing.T) {
	s := HealthStatus{Healthy: true, MemPercent: MemCriticalPercent}
	applyThresholds(&s)

	if s.Healthy {
		t.Error("expected Healthy=false when memory is at threshold")
	}
	if len(s.Alerts) != 1 {
		t.Errorf("expected 1 alert, got %d: %v", len(s.Alerts), s.Alerts)
	}
	if !strings.Contains(s.Alerts[0], "Memory") {
		t.Errorf("expected memory alert, got: %q", s.Alerts[0])
	}
}

// TestApplyThresholds_DiskCritical verifies the disk threshold fires correctly.
func TestApplyThresholds_DiskCritical(t *testing.T) {
	s := HealthStatus{Healthy: true, DiskPercent: DiskCriticalPercent}
	applyThresholds(&s)

	if s.Healthy {
		t.Error("expected Healthy=false when disk is at threshold")
	}
	if len(s.Alerts) != 1 {
		t.Errorf("expected 1 alert, got %d: %v", len(s.Alerts), s.Alerts)
	}
	if !strings.Contains(s.Alerts[0], "Disk") {
		t.Errorf("expected disk alert, got: %q", s.Alerts[0])
	}
}

// TestApplyThresholds_TempCritical verifies the temperature threshold fires correctly.
func TestApplyThresholds_TempCritical(t *testing.T) {
	s := HealthStatus{Healthy: true, TemperatureCelsius: TempCriticalCelsius}
	applyThresholds(&s)

	if s.Healthy {
		t.Error("expected Healthy=false when temperature is at threshold")
	}
	if len(s.Alerts) != 1 {
		t.Errorf("expected 1 alert, got %d: %v", len(s.Alerts), s.Alerts)
	}
	if !strings.Contains(s.Alerts[0], "Temperature") {
		t.Errorf("expected temperature alert, got: %q", s.Alerts[0])
	}
}

// TestApplyThresholds_MultipleAlerts verifies that all three thresholds can
// fire simultaneously and each produces its own alert.
func TestApplyThresholds_MultipleAlerts(t *testing.T) {
	s := HealthStatus{
		Healthy:            true,
		MemPercent:         MemCriticalPercent,
		DiskPercent:        DiskCriticalPercent,
		TemperatureCelsius: TempCriticalCelsius,
	}
	applyThresholds(&s)

	if s.Healthy {
		t.Error("expected Healthy=false when all thresholds breached")
	}
	if len(s.Alerts) != 3 {
		t.Errorf("expected 3 alerts, got %d: %v", len(s.Alerts), s.Alerts)
	}
}

// TestApplyThresholds_JustBelowThresholds verifies that values one unit below
// each threshold do not trigger alerts — the boundary condition.
func TestApplyThresholds_JustBelowThresholds(t *testing.T) {
	s := HealthStatus{
		Healthy:            true,
		MemPercent:         MemCriticalPercent - 0.1,
		DiskPercent:        DiskCriticalPercent - 0.1,
		TemperatureCelsius: TempCriticalCelsius - 0.1,
	}
	applyThresholds(&s)

	if !s.Healthy {
		t.Errorf("expected Healthy=true just below thresholds, alerts: %v", s.Alerts)
	}
	if len(s.Alerts) != 0 {
		t.Errorf("expected no alerts just below thresholds, got: %v", s.Alerts)
	}
}

// TestApplyThresholds_DoesNotClearExistingAlerts verifies that applyThresholds
// appends alerts rather than replacing them — callers (e.g. SMART checks) may
// have already added their own before calling applyThresholds.
func TestApplyThresholds_DoesNotClearExistingAlerts(t *testing.T) {
	existing := "SMART: drive /dev/sda pre-failure"
	s := HealthStatus{
		Healthy: false,
		Alerts:  []string{existing},
		// All hardware metrics below threshold — only SMART alert should survive.
		MemPercent:         10.0,
		DiskPercent:        20.0,
		TemperatureCelsius: 40.0,
	}
	applyThresholds(&s)

	if len(s.Alerts) != 1 {
		t.Errorf("expected 1 alert (existing), got %d: %v", len(s.Alerts), s.Alerts)
	}
	if s.Alerts[0] != existing {
		t.Errorf("existing alert was modified: got %q, want %q", s.Alerts[0], existing)
	}
}

// TestThresholdConstants verifies the threshold constants are in sensible ranges.
func TestThresholdConstants(t *testing.T) {
	if TempCriticalCelsius <= 0 || TempCriticalCelsius > 120 {
		t.Errorf("TempCriticalCelsius %v out of sane range (0, 120]", TempCriticalCelsius)
	}
	if CPUCriticalPercent <= 0 || CPUCriticalPercent > 100 {
		t.Errorf("CPUCriticalPercent %v out of range (0, 100]", CPUCriticalPercent)
	}
	if MemCriticalPercent <= 0 || MemCriticalPercent > 100 {
		t.Errorf("MemCriticalPercent %v out of range (0, 100]", MemCriticalPercent)
	}
	if DiskCriticalPercent <= 0 || DiskCriticalPercent > 100 {
		t.Errorf("DiskCriticalPercent %v out of range (0, 100]", DiskCriticalPercent)
	}
}
