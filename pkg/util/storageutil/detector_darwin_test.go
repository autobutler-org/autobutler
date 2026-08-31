package storageutil

import (
	"math"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// TestDetectDevicesMatchesDfCapacity checks the reported usage against df's own
// Capacity column on this machine: APFS volumes share a container, so a
// per-volume Used ratio understates real fullness (issue #1711).
func TestDetectDevicesMatchesDfCapacity(t *testing.T) {
	devices, err := NewDetector().DetectDevices()
	if err != nil {
		t.Skipf("df unavailable: %v", err)
	}

	seen := map[string]bool{}
	for _, device := range devices {
		if device.TotalBytes == 0 {
			continue
		}
		container := (&detector{}).getContainerID(device.DevicePath)
		if seen[container] {
			t.Errorf("container %s reported twice (%s)", container, device.DevicePath)
		}
		seen[container] = true

		if device.UsedBytes+device.AvailableBytes != device.TotalBytes {
			t.Errorf("%s: used (%d) + avail (%d) != total (%d)", device.MountPoint,
				device.UsedBytes, device.AvailableBytes, device.TotalBytes)
		}

		want := dfCapacity(t, device.MountPoint)
		got := float64(device.UsedBytes) / float64(device.TotalBytes) * 100
		if math.Abs(got-want) > 1.5 {
			t.Errorf("%s: reported %.1f%% used, df says %.0f%%", device.MountPoint, got, want)
		}
	}
}

func dfCapacity(t *testing.T, mountPoint string) float64 {
	t.Helper()
	out, err := exec.Command("df", "-k", mountPoint).Output()
	if err != nil {
		t.Fatalf("df %s: %v", mountPoint, err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	fields := strings.Fields(lines[len(lines)-1])
	pct, err := strconv.ParseFloat(strings.TrimSuffix(fields[4], "%"), 64)
	if err != nil {
		t.Fatalf("parse capacity %q: %v", fields[4], err)
	}
	return pct
}
