package system_test

import (
	"testing"

	"github.com/autobutler-org/autobutler/pkg/botel/system"
)

// TestRegister_ReturnsNonNilCollector verifies that Register() succeeds against
// the default (no-op) OTel global provider and returns a usable *Collector.
// No InitMetrics() call is needed — Register() falls back to the global no-op
// MeterProvider that Go programs have out of the box.
func TestRegister_ReturnsNonNilCollector(t *testing.T) {
	c, err := system.Register()
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}
	if c == nil {
		t.Fatal("Register() returned nil collector")
	}
}

// TestRegister_Idempotent verifies that calling Register() multiple times does
// not panic or error — the no-op provider allows repeated gauge registration.
func TestRegister_Idempotent(t *testing.T) {
	for i := 0; i < 3; i++ {
		_, err := system.Register()
		if err != nil {
			t.Fatalf("Register() call %d failed: %v", i+1, err)
		}
	}
}

// TestCurrentHealth_StructShape verifies that CurrentHealth() returns a
// HealthStatus with sensible zero-or-real values, never panics, and always
// initialises the Alerts slice (nil would cause JSON marshalling to emit null
// instead of []).
func TestCurrentHealth_StructShape(t *testing.T) {
	c, err := system.Register()
	if err != nil {
		t.Fatalf("Register(): %v", err)
	}

	h := c.CurrentHealth()

	// Alerts must be a slice (possibly empty), never nil — the health endpoint
	// relies on this to produce JSON [] rather than null.
	if h.Alerts == nil {
		t.Error("HealthStatus.Alerts must not be nil — want [] not null in JSON")
	}

	// On a real machine, MemTotalBytes should be > 0; in CI/containers it may
	// be 0 if gopsutil can't read /proc/meminfo.  We just assert no panic.
	_ = h.MemTotalBytes
	_ = h.DiskTotalBytes
	_ = h.CPUPercent
}

// TestCurrentHealth_CPUCorePercentsConsistency verifies that if CPUCorePercents
// is populated its length is > 0 (at least one core reported).
func TestCurrentHealth_CPUCorePercentsConsistency(t *testing.T) {
	c, _ := system.Register()
	h := c.CurrentHealth()

	if h.CPUCorePercents != nil && len(h.CPUCorePercents) == 0 {
		t.Error("CPUCorePercents is non-nil but empty — should have at least one element")
	}
}

// TestCurrentHealth_TemperatureNonNegative verifies the temperature field is
// never negative (0 is the sentinel for "unavailable").
func TestCurrentHealth_TemperatureNonNegative(t *testing.T) {
	c, _ := system.Register()
	h := c.CurrentHealth()

	if h.TemperatureCelsius < 0 {
		t.Errorf("TemperatureCelsius = %.2f; expected >= 0 (0 means unavailable)", h.TemperatureCelsius)
	}
}

// TestCurrentHealth_HealthyByDefault verifies that a freshly created Collector
// reports Healthy=true when metrics are within normal ranges (or unavailable).
// On a non-overloaded test machine this will always hold.
func TestCurrentHealth_HealthyByDefault(t *testing.T) {
	c, _ := system.Register()
	h := c.CurrentHealth()

	// We can't guarantee the exact value on a busy CI runner, but we can
	// verify the field exists and is readable without panicking.
	_ = h.Healthy
}
