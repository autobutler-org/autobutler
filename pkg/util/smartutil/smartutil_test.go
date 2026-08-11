package smartutil_test

import (
	"testing"

	"github.com/autobutler-org/autobutler/pkg/util/smartutil"
)

// smartctl -j -a /dev/sda JSON excerpt with a healthy drive.
const healthyJSON = `{
  "model_name": "Samsung SSD 860 EVO 1TB",
  "serial_number": "S3Z2NX0K123456",
  "smart_status": {"passed": true},
  "temperature": {"current": 32},
  "ata_smart_attributes": {
    "table": [
      {"id": 5,   "raw": {"value": 0,    "string": "0"}},
      {"id": 9,   "raw": {"value": 4231, "string": "4231"}},
      {"id": 197, "raw": {"value": 0,    "string": "0"}},
      {"id": 198, "raw": {"value": 0,    "string": "0"}}
    ]
  }
}`

// smartctl -j -a /dev/sdb JSON excerpt with pre-failure attributes set.
const failingJSON = `{
  "model_name": "WDC WD5000AAKX-001CA0",
  "serial_number": "WD-WCAYUE007890",
  "smart_status": {"passed": false},
  "temperature": {"current": 51},
  "ata_smart_attributes": {
    "table": [
      {"id": 5,   "raw": {"value": 17,  "string": "17"}},
      {"id": 9,   "raw": {"value": 31042, "string": "31042"}},
      {"id": 197, "raw": {"value": 3,   "string": "3"}},
      {"id": 198, "raw": {"value": 1,   "string": "1"}}
    ]
  }
}`

// TestParseHealthyDrive verifies that a healthy SMART JSON response maps to
// the expected DriveHealth fields with no alerts.
func TestParseHealthyDrive(t *testing.T) {
	h := smartutil.ParseJSON([]byte(healthyJSON), "/dev/sda")

	if !h.Healthy {
		t.Error("expected Healthy=true for PASSED drive")
	}
	if h.ModelName != "Samsung SSD 860 EVO 1TB" {
		t.Errorf("ModelName: got %q", h.ModelName)
	}
	if h.TemperatureCelsius != 32 {
		t.Errorf("Temperature: got %v, want 32", h.TemperatureCelsius)
	}
	if h.PowerOnHours != 4231 {
		t.Errorf("PowerOnHours: got %v, want 4231", h.PowerOnHours)
	}
	if h.ReallocatedSectors != 0 {
		t.Errorf("ReallocatedSectors: got %v, want 0", h.ReallocatedSectors)
	}
	if len(h.Alerts) != 0 {
		t.Errorf("Alerts: expected none, got %v", h.Alerts)
	}
}

// TestParseFailingDrive verifies that pre-failure attributes and a FAILED
// SMART status produce the expected alerts.
func TestParseFailingDrive(t *testing.T) {
	h := smartutil.ParseJSON([]byte(failingJSON), "/dev/sdb")

	if h.Healthy {
		t.Error("expected Healthy=false for FAILED drive")
	}
	if h.ReallocatedSectors != 17 {
		t.Errorf("ReallocatedSectors: got %v, want 17", h.ReallocatedSectors)
	}
	if h.PendingSectors != 3 {
		t.Errorf("PendingSectors: got %v, want 3", h.PendingSectors)
	}
	if h.UncorrectableErrors != 1 {
		t.Errorf("UncorrectableErrors: got %v, want 1", h.UncorrectableErrors)
	}
	if h.PowerOnHours != 31042 {
		t.Errorf("PowerOnHours: got %v, want 31042", h.PowerOnHours)
	}
	if len(h.Alerts) == 0 {
		t.Error("expected alerts for failing drive, got none")
	}
	// Should have: FAILED assessment + reallocated sectors + pending + uncorrectable = 4 alerts.
	if len(h.Alerts) != 4 {
		t.Errorf("expected 4 alerts, got %d: %v", len(h.Alerts), h.Alerts)
	}
}

// TestParseEmptyJSON verifies that an empty or unparseable input returns a
// zero DriveHealth without panicking.
func TestParseEmptyJSON(t *testing.T) {
	h := smartutil.ParseJSON([]byte("{}"), "/dev/sdc")
	if h.Healthy {
		t.Error("expected Healthy=false for empty JSON (no passed:true)")
	}
	if len(h.Alerts) != 0 {
		t.Errorf("expected no alerts for empty JSON, got %v", h.Alerts)
	}
}

// TestParseInvalidJSON verifies graceful handling of non-JSON input.
func TestParseInvalidJSON(t *testing.T) {
	h := smartutil.ParseJSON([]byte("not json"), "/dev/sdd")
	if h.Device != "/dev/sdd" {
		t.Errorf("Device not set: got %q", h.Device)
	}
}

// TestAvailable just ensures the function doesn't panic.
func TestAvailable(t *testing.T) {
	// We don't assert true/false — smartctl may or may not be installed.
	_ = smartutil.Available()
}
