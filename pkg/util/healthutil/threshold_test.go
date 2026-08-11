package healthutil

import (
	"strings"
	"testing"
	"time"
)

// --- CPU: sustained-load rule ---

func TestApplyCPUThreshold_BelowLimit_ClearsMarker(t *testing.T) {
	status := HealthStatus{Healthy: true}
	past := time.Now().Add(-time.Hour)
	got := applyCPUThreshold(&status, CPUCriticalPercent-1, &past, time.Now())
	if got != nil {
		t.Error("dropping below the limit should clear the high-since marker")
	}
	if !status.Healthy || len(status.Alerts) != 0 {
		t.Errorf("below limit should not alert; got healthy=%v alerts=%v", status.Healthy, status.Alerts)
	}
}

func TestApplyCPUThreshold_FirstBreach_StartsClockWithoutAlerting(t *testing.T) {
	status := HealthStatus{Healthy: true}
	now := time.Now()
	got := applyCPUThreshold(&status, CPUCriticalPercent, nil, now)
	if got == nil || !got.Equal(now) {
		t.Errorf("first breach should start the clock at now; got %v", got)
	}
	if !status.Healthy || len(status.Alerts) != 0 {
		t.Error("a single spike must not mark the host unhealthy")
	}
}

func TestApplyCPUThreshold_BreachShorterThanWindow_DoesNotAlert(t *testing.T) {
	status := HealthStatus{Healthy: true}
	now := time.Now()
	since := now.Add(-(CPUSustainedDuration / 2))
	applyCPUThreshold(&status, 99, &since, now)
	if !status.Healthy || len(status.Alerts) != 0 {
		t.Errorf("breach shorter than %s must not alert; got %v", CPUSustainedDuration, status.Alerts)
	}
}

func TestApplyCPUThreshold_SustainedBreach_Alerts(t *testing.T) {
	status := HealthStatus{Healthy: true}
	now := time.Now()
	since := now.Add(-(CPUSustainedDuration + time.Second))
	got := applyCPUThreshold(&status, 99, &since, now)
	if status.Healthy {
		t.Error("sustained breach should mark unhealthy")
	}
	if len(status.Alerts) != 1 || !strings.Contains(status.Alerts[0], "CPU sustained") {
		t.Errorf("expected a CPU sustained alert, got %v", status.Alerts)
	}
	if got != &since {
		t.Error("marker should persist while CPU stays high")
	}
}

// --- Memory / Disk / Temp: simple boundary rules ---

func TestApplyMemThreshold(t *testing.T) {
	tests := []struct {
		name      string
		pct       float64
		wantAlert bool
	}{
		{"well below", 50, false},
		{"just below", MemCriticalPercent - 0.1, false},
		{"exactly at limit", MemCriticalPercent, true},
		{"above", MemCriticalPercent + 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := HealthStatus{Healthy: true}
			applyMemThreshold(&status, tt.pct)
			if gotAlert := len(status.Alerts) > 0; gotAlert != tt.wantAlert {
				t.Errorf("mem %.1f%%: alert=%v, want %v", tt.pct, gotAlert, tt.wantAlert)
			}
			if status.Healthy == tt.wantAlert {
				t.Errorf("mem %.1f%%: healthy=%v contradicts alert=%v", tt.pct, status.Healthy, tt.wantAlert)
			}
		})
	}
}

func TestApplyDiskThreshold(t *testing.T) {
	tests := []struct {
		name      string
		pct       float64
		wantAlert bool
	}{
		{"well below", 10, false},
		{"just below", DiskCriticalPercent - 0.1, false},
		{"exactly at limit", DiskCriticalPercent, true},
		{"nearly full", 99.9, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := HealthStatus{Healthy: true}
			applyDiskThreshold(&status, tt.pct)
			if gotAlert := len(status.Alerts) > 0; gotAlert != tt.wantAlert {
				t.Errorf("disk %.1f%%: alert=%v, want %v", tt.pct, gotAlert, tt.wantAlert)
			}
		})
	}
}

func TestApplyTempThreshold(t *testing.T) {
	tests := []struct {
		name      string
		temp      float64
		wantAlert bool
	}{
		{"cool", 40, false},
		{"warm but safe", TempCriticalCelsius - 0.1, false},
		{"at throttle point", TempCriticalCelsius, true},
		{"overheating", TempCriticalCelsius + 10, true},
		{"sensor absent reads zero", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := HealthStatus{Healthy: true}
			applyTempThreshold(&status, tt.temp)
			if gotAlert := len(status.Alerts) > 0; gotAlert != tt.wantAlert {
				t.Errorf("temp %.1f°C: alert=%v, want %v", tt.temp, gotAlert, tt.wantAlert)
			}
		})
	}
}

// TestApplyThresholds_AlertsAccumulate verifies that simultaneous breaches all
// report, rather than the last one overwriting the others.
func TestApplyThresholds_AlertsAccumulate(t *testing.T) {
	status := HealthStatus{Healthy: true}
	applyMemThreshold(&status, 99)
	applyDiskThreshold(&status, 99)
	applyTempThreshold(&status, 99)
	if len(status.Alerts) != 3 {
		t.Errorf("expected 3 accumulated alerts, got %d: %v", len(status.Alerts), status.Alerts)
	}
	if status.Healthy {
		t.Error("multiple breaches must leave the host unhealthy")
	}
}
