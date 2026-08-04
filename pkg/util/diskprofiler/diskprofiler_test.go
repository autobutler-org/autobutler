package diskprofiler_test

import (
	"testing"

	"github.com/autobutler-org/autobutler/pkg/util/diskprofiler"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		mbps float64
		want diskprofiler.Class
	}{
		{500, diskprofiler.ClassFast},
		{150, diskprofiler.ClassFast},
		{149, diskprofiler.ClassMedium},
		{60, diskprofiler.ClassMedium},
		{30, diskprofiler.ClassMedium},
		{29, diskprofiler.ClassSlow},
		{10, diskprofiler.ClassSlow},
		{0, diskprofiler.ClassSlow},
	}
	for _, tt := range tests {
		got := diskprofiler.Classify(tt.mbps)
		if got != tt.want {
			t.Errorf("Classify(%.0f): got %q, want %q", tt.mbps, got, tt.want)
		}
	}
}

func TestRecommendConcurrency(t *testing.T) {
	if diskprofiler.RecommendConcurrency(diskprofiler.ClassFast) <= diskprofiler.RecommendConcurrency(diskprofiler.ClassMedium) {
		t.Error("fast should recommend more concurrency than medium")
	}
	if diskprofiler.RecommendConcurrency(diskprofiler.ClassMedium) <= diskprofiler.RecommendConcurrency(diskprofiler.ClassSlow) {
		t.Error("medium should recommend more concurrency than slow")
	}
}

// TestProfile runs a real disk profile against a temp directory. On any
// reasonable disk this should take <500 ms and return a positive read speed.
// This test is skipped in short mode to keep CI fast.
func TestProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping disk IO benchmark in short mode")
	}

	dir := t.TempDir()
	result, err := diskprofiler.Profile(dir)
	if err != nil {
		t.Fatalf("Profile(%q): %v", dir, err)
	}

	if result.SeqReadMBps <= 0 {
		t.Errorf("SeqReadMBps should be positive, got %f", result.SeqReadMBps)
	}
	if result.RandReadLatencyMs < 0 {
		t.Errorf("RandReadLatencyMs should be non-negative, got %f", result.RandReadLatencyMs)
	}
	if result.Class == "" {
		t.Error("Class should not be empty")
	}
	if result.RecommendedConcurrency <= 0 {
		t.Errorf("RecommendedConcurrency should be positive, got %d", result.RecommendedConcurrency)
	}

	// Verify Class matches SeqReadMBps.
	if got := diskprofiler.Classify(result.SeqReadMBps); got != result.Class {
		t.Errorf("Class mismatch: result.Class=%q but Classify(%.2f)=%q", result.Class, result.SeqReadMBps, got)
	}

	t.Logf("Profile result: seq=%.1f MB/s rand_latency=%.2f ms class=%s concurrency=%d",
		result.SeqReadMBps, result.RandReadLatencyMs, result.Class, result.RecommendedConcurrency)
}

// TestProfile_InvalidDir verifies an error is returned for a non-existent dir.
func TestProfile_InvalidDir(t *testing.T) {
	_, err := diskprofiler.Profile("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for invalid directory, got nil")
	}
}

func TestRecommendCleanupTimeout(t *testing.T) {
	fast := diskprofiler.RecommendCleanupTimeout(diskprofiler.ClassFast)
	medium := diskprofiler.RecommendCleanupTimeout(diskprofiler.ClassMedium)
	slow := diskprofiler.RecommendCleanupTimeout(diskprofiler.ClassSlow)

	if fast >= medium {
		t.Errorf("fast timeout (%v) should be less than medium (%v)", fast, medium)
	}
	if medium >= slow {
		t.Errorf("medium timeout (%v) should be less than slow (%v)", medium, slow)
	}
	if slow <= 0 {
		t.Error("slow timeout should be positive")
	}
}
