// Package smartutil queries S.M.A.R.T. drive health data via smartctl(8).
//
// If smartctl is not installed or the caller lacks permission, all functions
// return a zero/empty result rather than an error — callers treat absence as
// "monitoring unavailable" rather than a hard failure.
//
// Requires smartmontools ≥ 7.0 (for reliable -j JSON output).
package smartutil

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// DriveHealth summarises the S.M.A.R.T. status for one physical drive.
type DriveHealth struct {
	// Device is the OS path, e.g. "/dev/sda".
	Device string `json:"device"`

	// Healthy is true when the overall SMART assessment is PASSED.
	Healthy bool `json:"healthy"`

	// TemperatureCelsius is the current drive temperature (0 if unavailable).
	TemperatureCelsius float64 `json:"temperatureCelsius,omitempty"`

	// PowerOnHours is the total drive power-on time (0 if unavailable).
	PowerOnHours int64 `json:"powerOnHours,omitempty"`

	// ReallocatedSectors is the count of reallocated sectors (non-zero = bad
	// blocks remapped — a leading indicator of imminent failure).
	ReallocatedSectors int64 `json:"reallocatedSectors,omitempty"`

	// PendingSectors is the count of sectors pending reallocation.
	PendingSectors int64 `json:"pendingSectors,omitempty"`

	// UncorrectableErrors is the count of offline uncorrectable errors.
	UncorrectableErrors int64 `json:"uncorrectableErrors,omitempty"`

	// ModelName is the drive model string returned by smartctl.
	ModelName string `json:"modelName,omitempty"`

	// SerialNumber is the drive serial number.
	SerialNumber string `json:"serialNumber,omitempty"`

	// Alerts contains human-readable warnings when pre-failure attributes are
	// non-zero or the overall SMART assessment is FAILED/UNKNOWN.
	Alerts []string `json:"alerts,omitempty"`
}

// Available returns true if smartctl(8) is found in PATH.
func Available() bool {
	_, err := exec.LookPath("smartctl")
	return err == nil
}

// Query runs smartctl -j (JSON output) on the given device path and parses
// the result into a DriveHealth. Returns an empty DriveHealth and no error
// when smartctl is unavailable or the device does not support SMART.
func Query(device string) (DriveHealth, error) {
	h := DriveHealth{Device: device}

	// #nosec G204 — device is a validated path from the storage detector, not
	// user input. It is always an absolute device node (e.g. /dev/sda).
	out, err := exec.Command("smartctl", "-j", "-a", device).Output()
	if err != nil {
		// smartctl returns non-zero on some warning conditions but still emits
		// JSON. Try to parse what we got before giving up.
		if len(out) == 0 {
			return h, nil
		}
	}

	return ParseJSON(out, device), nil
}

// ParseJSON parses the JSON output of `smartctl -j -a` into a DriveHealth.
// Exported for use in tests that inject fixture JSON without running smartctl.
func ParseJSON(data []byte, device string) DriveHealth {
	h := DriveHealth{Device: device}

	var raw smartctlJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return h // unparseable — treat as unavailable
	}

	h.ModelName = raw.ModelName
	h.SerialNumber = raw.SerialNumber
	h.Healthy = raw.SmartStatus.Passed

	if raw.Temperature.Current != 0 {
		h.TemperatureCelsius = float64(raw.Temperature.Current)
	}

	for _, attr := range raw.AtaSmartAttributes.Table {
		switch attr.ID {
		case 5: // Reallocated_Sector_Ct
			h.ReallocatedSectors = attr.Raw.Value
		case 197: // Current_Pending_Sector
			h.PendingSectors = attr.Raw.Value
		case 198: // Offline_Uncorrectable
			h.UncorrectableErrors = attr.Raw.Value
		case 9: // Power_On_Hours
			h.PowerOnHours = attr.Raw.Value
		}
	}

	// Build alerts. Only surface a FAILED assessment alert when smartctl
	// returned real device data (ModelName set). An empty model means the
	// device doesn't support SMART or didn't respond — no alert in that case.
	if !h.Healthy && h.ModelName != "" {
		h.Alerts = append(h.Alerts, "SMART assessment FAILED — drive may be failing, back up now")
	}
	if h.ReallocatedSectors > 0 {
		h.Alerts = append(h.Alerts,
			formatAttrAlert("reallocated sectors", h.ReallocatedSectors, "backing store may be failing"))
	}
	if h.PendingSectors > 0 {
		h.Alerts = append(h.Alerts,
			formatAttrAlert("pending sectors", h.PendingSectors, "drive has unstable sectors"))
	}
	if h.UncorrectableErrors > 0 {
		h.Alerts = append(h.Alerts,
			formatAttrAlert("uncorrectable errors", h.UncorrectableErrors, "data may be at risk"))
	}

	return h
}

// QueryDevices queries each device in the provided list and returns results
// for all devices that responded. Devices that are unavailable (no SMART
// support, permission denied, etc.) are silently omitted.
func QueryDevices(devices []string) []DriveHealth {
	if !Available() {
		return nil
	}
	out := make([]DriveHealth, 0, len(devices))
	for _, d := range devices {
		h, _ := Query(d)
		if h.ModelName != "" || h.ReallocatedSectors != 0 || h.PowerOnHours != 0 {
			// Only include drives that returned meaningful data.
			out = append(out, h)
		}
	}
	return out
}

// ProbeTimeout is the maximum time allowed for a single smartctl invocation.
// Adjust if device enumeration is slow on a specific host.
const ProbeTimeout = 10 * time.Second

func formatAttrAlert(attr string, count int64, detail string) string {
	return fmt.Sprintf("SMART: %d %s (%s)", count, attr, detail)
}

// ─── internal JSON schema ────────────────────────────────────────────────────

type smartctlJSON struct {
	ModelName          string      `json:"model_name"`
	SerialNumber       string      `json:"serial_number"`
	SmartStatus        smartStatus `json:"smart_status"`
	Temperature        temperature `json:"temperature"`
	AtaSmartAttributes smartAttrs  `json:"ata_smart_attributes"`
}

type smartStatus struct {
	Passed bool `json:"passed"`
}

type temperature struct {
	Current int `json:"current"`
}

type smartAttrs struct {
	Table []smartAttr `json:"table"`
}

type smartAttr struct {
	ID  int      `json:"id"`
	Raw smartRaw `json:"raw"`
}

type smartRaw struct {
	Value  int64  `json:"value"`
	String string `json:"string"`
}
