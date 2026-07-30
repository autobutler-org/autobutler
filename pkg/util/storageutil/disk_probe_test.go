package storageutil_test

import (
	"testing"

	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
)

func TestProbeDisk_ReturnsResult(t *testing.T) {
	dir := t.TempDir()
	result := storageutil.ProbeDisk(dir)

	// On any real filesystem the probe should succeed and give a valid speed.
	if result.SpeedClass == storageutil.DiskSpeedUnknown {
		t.Errorf("expected a classified result, got DiskSpeedUnknown; seqReadMBps=%.2f", result.SeqReadMBps)
	}
	if result.SeqReadMBps <= 0 {
		t.Errorf("expected positive SeqReadMBps, got %.2f", result.SeqReadMBps)
	}
	if result.RandReadLatencyMs < 0 {
		t.Errorf("unexpected negative RandReadLatencyMs: %.2f", result.RandReadLatencyMs)
	}
}

func TestProbeDisk_InvalidDir(t *testing.T) {
	result := storageutil.ProbeDisk("/nonexistent/path/that/does/not/exist")
	// Should return unknown gracefully, not panic.
	if result.SpeedClass != storageutil.DiskSpeedUnknown {
		t.Errorf("expected DiskSpeedUnknown for invalid dir, got %q", result.SpeedClass)
	}
}

func TestClassifyDisk_Thresholds(t *testing.T) {
	cases := []struct {
		seqMBps float64
		want    storageutil.DiskSpeedClass
	}{
		{250, storageutil.DiskSpeedFast},
		{200, storageutil.DiskSpeedFast},
		{199, storageutil.DiskSpeedMedium},
		{50, storageutil.DiskSpeedMedium},
		{20, storageutil.DiskSpeedMedium},
		{19.9, storageutil.DiskSpeedSlow},
		{5, storageutil.DiskSpeedSlow},
		{0, storageutil.DiskSpeedSlow},
	}
	for _, tc := range cases {
		// Use ProbeDisk indirectly via known threshold values — we verify the
		// classification constants match the documented thresholds.
		got := storageutil.ClassifyDiskSpeed(tc.seqMBps)
		if got != tc.want {
			t.Errorf("ClassifyDiskSpeed(%.1f): got %q, want %q", tc.seqMBps, got, tc.want)
		}
	}
}

func TestDeviceStatus_HasDiskProbeField(t *testing.T) {
	// Compile-time check: DiskProbe field exists and is the right type.
	ds := storageutil.DeviceStatus{}
	if ds.DiskProbe != nil {
		t.Error("expected nil DiskProbe on zero-value DeviceStatus")
	}
}
