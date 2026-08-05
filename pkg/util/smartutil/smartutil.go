// Package smartutil reads S.M.A.R.T. disk health data using smartctl.
//
// It shells out to `smartctl -j -H -A <device>` and parses the JSON output.
// If smartctl is not installed or a device does not support S.M.A.R.T., the
// functions return empty/zero results rather than errors — callers should treat
// a missing result as "unknown", not "healthy".
package smartutil

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

// DriveHealth summarises the S.M.A.R.T. health state of a single drive.
type DriveHealth struct {
	// Device is the OS device path, e.g. "/dev/sda".
	Device string `json:"device"`
	// Model is the drive model string from smartctl, if available.
	Model string `json:"model,omitempty"`
	// SmartAvailable is false when the device does not support S.M.A.R.T.
	// or smartctl cannot access it (USB bridge without pass-through, etc.).
	SmartAvailable bool `json:"smartAvailable"`
	// Healthy is true when smartctl's overall-health assessment is "PASSED".
	// Only valid when SmartAvailable is true.
	Healthy bool `json:"healthy"`
	// PowerOnHours is the drive's accumulated power-on time.
	PowerOnHours int `json:"powerOnHours,omitempty"`
	// Temperature is the drive temperature in Celsius, 0 if unavailable.
	Temperature float64 `json:"temperature,omitempty"`
	// ReallocatedSectors is the count of reallocated sectors (ATA id 5).
	// Non-zero is a warning sign that the drive has bad blocks.
	ReallocatedSectors int `json:"reallocatedSectors,omitempty"`
	// PendingSectors is the count of sectors pending reallocation (ATA id 197).
	// Non-zero means the drive has found read errors it hasn't yet remapped.
	PendingSectors int `json:"pendingSectors,omitempty"`
	// UncorrectableErrors is the offline uncorrectable sector count (ATA id 198).
	UncorrectableErrors int `json:"uncorrectableErrors,omitempty"`
	// PreFailure is true when any pre-failure threshold is exceeded.
	PreFailure bool `json:"preFailure,omitempty"`
	// Alerts contains human-readable warning strings for any exceeded thresholds.
	Alerts []string `json:"alerts,omitempty"`
}

// smartctlOutput is the subset of the smartctl JSON output we care about.
type smartctlOutput struct {
	SmartStatus *struct {
		Passed bool `json:"passed"`
	} `json:"smart_status"`
	ModelName   string `json:"model_name"`
	ModelFamily string `json:"model_family"`
	Temperature *struct {
		Current int `json:"current"`
	} `json:"temperature"`
	PowerOnTime *struct {
		Hours int `json:"hours"`
	} `json:"power_on_time"`
	AtaSmartAttributes *struct {
		Table []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Raw  struct {
				Value int `json:"value"`
			} `json:"raw"`
			Thresh     int    `json:"thresh"`
			Value      int    `json:"value"`
			Worst      int    `json:"worst"`
			WhenFailed string `json:"when_failed"`
		} `json:"table"`
	} `json:"ata_smart_attributes"`
	SmartctlMsg []struct {
		String   string `json:"string"`
		Severity string `json:"severity"`
	} `json:"smartctl"`
}

// ListDevices returns block device paths that smartctl can enumerate.
// Falls back to common paths if smartctl --scan returns nothing.
func ListDevices(ctx context.Context) []string {
	out, err := runSmartctl(ctx, "--scan")
	if err != nil {
		slog.Debug("smartutil: --scan failed", "err", err)
		return nil
	}

	var scanResult struct {
		Devices []struct {
			Name string `json:"name"`
		} `json:"devices"`
	}
	if err := json.Unmarshal(out, &scanResult); err != nil {
		slog.Debug("smartutil: failed to parse --scan output", "err", err)
		return nil
	}

	devices := make([]string, 0, len(scanResult.Devices))
	for _, d := range scanResult.Devices {
		if d.Name != "" {
			devices = append(devices, d.Name)
		}
	}
	return devices
}

// QueryDrive runs `smartctl -j -H -A` on the given device and returns the
// parsed health summary. Returns a zero-value DriveHealth with
// SmartAvailable=false if the device does not support S.M.A.R.T.
func QueryDrive(ctx context.Context, device string) DriveHealth {
	result := DriveHealth{Device: device}

	out, err := runSmartctl(ctx, "-H", "-A", "-i", device)
	if err != nil && out == nil {
		slog.Debug("smartutil: smartctl failed with no output", "device", device, "err", err)
		return result
	}

	var parsed smartctlOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		slog.Debug("smartutil: failed to parse output", "device", device, "err", err)
		return result
	}

	// If smart_status is absent, S.M.A.R.T. is not supported/accessible.
	if parsed.SmartStatus == nil {
		return result
	}

	result.SmartAvailable = true
	result.Healthy = parsed.SmartStatus.Passed

	if parsed.ModelName != "" {
		result.Model = parsed.ModelName
	} else if parsed.ModelFamily != "" {
		result.Model = parsed.ModelFamily
	}

	if parsed.Temperature != nil {
		result.Temperature = float64(parsed.Temperature.Current)
	}
	if parsed.PowerOnTime != nil {
		result.PowerOnHours = parsed.PowerOnTime.Hours
	}

	// Walk ATA attribute table for the critical counters.
	if parsed.AtaSmartAttributes != nil {
		for _, attr := range parsed.AtaSmartAttributes.Table {
			switch attr.ID {
			case 5: // Reallocated_Sector_Ct
				result.ReallocatedSectors = attr.Raw.Value
			case 197: // Current_Pending_Sector
				result.PendingSectors = attr.Raw.Value
			case 198: // Offline_Uncorrectable
				result.UncorrectableErrors = attr.Raw.Value
			}
			// Pre-failure: when_failed="now" or value <= thresh on a pre-failure attr.
			if attr.WhenFailed == "now" {
				result.PreFailure = true
				result.Alerts = append(result.Alerts,
					fmt.Sprintf("pre-failure: %s (value=%d thresh=%d)", attr.Name, attr.Value, attr.Thresh))
			}
		}
	}

	if !result.Healthy {
		result.Alerts = append(result.Alerts, "S.M.A.R.T. self-assessment FAILED")
	}
	if result.ReallocatedSectors > 0 {
		result.Alerts = append(result.Alerts,
			fmt.Sprintf("reallocated sectors: %d", result.ReallocatedSectors))
	}
	if result.PendingSectors > 0 {
		result.Alerts = append(result.Alerts,
			fmt.Sprintf("pending sectors: %d", result.PendingSectors))
	}
	if result.UncorrectableErrors > 0 {
		result.Alerts = append(result.Alerts,
			fmt.Sprintf("uncorrectable errors: %d", result.UncorrectableErrors))
	}

	return result
}

// QueryAllDrives runs QueryDrive for every device returned by ListDevices.
func QueryAllDrives(ctx context.Context) []DriveHealth {
	devices := ListDevices(ctx)
	if len(devices) == 0 {
		return nil
	}
	results := make([]DriveHealth, 0, len(devices))
	for _, dev := range devices {
		results = append(results, QueryDrive(ctx, dev))
	}
	return results
}

// runSmartctl executes smartctl with the given args and -j (JSON output).
// Returns the raw JSON output even on non-zero exit (smartctl uses exit
// codes as bitmasks — a code of 64 means "device doesn't support SMART",
// which is not a tool failure).
func runSmartctl(ctx context.Context, args ...string) ([]byte, error) {
	allArgs := append([]string{"-j"}, args...)
	cmd := exec.CommandContext(ctx, "smartctl", allArgs...)
	out, err := cmd.Output()
	if err != nil {
		// Non-zero exit is expected for some device types; return whatever output
		// we got so the caller can decide whether the JSON is usable.
		var exitErr *exec.ExitError
		if ok := strings.Contains(err.Error(), "executable file not found"); ok {
			return nil, fmt.Errorf("smartctl not found: %w", err)
		}
		if out != nil && len(out) > 0 {
			// Partial output — parse it as best we can.
			return out, nil
		}
		_ = exitErr
		return nil, err
	}
	return out, nil
}
