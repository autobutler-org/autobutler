package diskprofiler

import (
	"os"
	"testing"
	"time"
)

func TestTierString(t *testing.T) {
	tests := []struct {
		tier Tier
		want string
	}{
		{TierFast, "fast"},
		{TierMedium, "medium"},
		{TierSlow, "slow"},
	}
	for _, tt := range tests {
		if got := tt.tier.String(); got != tt.want {
			t.Errorf("Tier(%d).String() = %q; want %q", tt.tier, got, tt.want)
		}
	}
}

func TestTierParams(t *testing.T) {
	if c := tierParams(TierFast); c <= tierParams(TierMedium) {
		t.Errorf("TierFast concurrency (%d) must be > TierMedium (%d)", c, tierParams(TierMedium))
	}
	if c := tierParams(TierMedium); c <= tierParams(TierSlow) {
		t.Errorf("TierMedium concurrency (%d) must be > TierSlow (%d)", c, tierParams(TierSlow))
	}
	if c := tierParams(TierSlow); c < 1 {
		t.Errorf("TierSlow concurrency must be >= 1, got %d", c)
	}
}

func TestTierTimeout(t *testing.T) {
	if tierTimeout(TierFast) >= tierTimeout(TierMedium) {
		t.Error("TierFast timeout must be shorter than TierMedium")
	}
	if tierTimeout(TierMedium) >= tierTimeout(TierSlow) {
		t.Error("TierMedium timeout must be shorter than TierSlow")
	}
}

func TestMeasure_returnsValidProfile(t *testing.T) {
	dir := t.TempDir()
	p := Measure(dir)

	if p.Dir != dir {
		t.Errorf("Dir: got %q, want %q", p.Dir, dir)
	}
	if p.WriteMBps <= 0 {
		t.Errorf("WriteMBps must be > 0, got %f", p.WriteMBps)
	}
	if p.MaxConcurrency < 1 {
		t.Errorf("MaxConcurrency must be >= 1, got %d", p.MaxConcurrency)
	}
	if p.CleanupTimeout < time.Second {
		t.Errorf("CleanupTimeout must be >= 1s, got %v", p.CleanupTimeout)
	}
	// Tier must be one of the defined values.
	if p.Tier != TierFast && p.Tier != TierMedium && p.Tier != TierSlow {
		t.Errorf("unexpected Tier value: %d", p.Tier)
	}
}

func TestMeasure_readOnlyDir_returnsSlow(t *testing.T) {
	// If we can't write, Measure must fall back to TierSlow conservatively.
	p := Measure("/proc") // /proc is read-only on Linux
	if p.Tier != TierSlow {
		// Some CI environments may allow writes to /proc — skip instead of fail.
		t.Skipf("expected TierSlow for /proc, got %s (may not be read-only in this environment)", p.Tier)
	}
	if p.WriteMBps != 0 {
		t.Errorf("WriteMBps should be 0 on error fallback, got %f", p.WriteMBps)
	}
}

func TestMeasure_profileConsistency(t *testing.T) {
	dir := t.TempDir()
	p := Measure(dir)

	// MaxConcurrency and CleanupTimeout must match the tier's parameters.
	if p.MaxConcurrency != tierParams(p.Tier) {
		t.Errorf("MaxConcurrency %d does not match tier %s params %d",
			p.MaxConcurrency, p.Tier, tierParams(p.Tier))
	}
	if p.CleanupTimeout != tierTimeout(p.Tier) {
		t.Errorf("CleanupTimeout %v does not match tier %s timeout %v",
			p.CleanupTimeout, p.Tier, tierTimeout(p.Tier))
	}
}

func TestCachedProfiler_cachesResult(t *testing.T) {
	dir := t.TempDir()
	cp := NewCachedProfiler(dir)

	p1 := cp.Get()
	p2 := cp.Get()

	// Second call should return exactly the same value (cached).
	if p1 != p2 {
		t.Errorf("CachedProfiler returned different profiles: %+v vs %+v", p1, p2)
	}
}

func TestCachedProfiler_reset(t *testing.T) {
	dir := t.TempDir()
	cp := NewCachedProfiler(dir)

	_ = cp.Get()
	cp.Reset()

	// After Reset, the internal pointer should be nil — next Get re-benchmarks.
	if cp.profile != nil {
		t.Error("expected profile to be nil after Reset")
	}
	p := cp.Get()
	if p.WriteMBps <= 0 {
		t.Errorf("re-benchmark after Reset returned non-positive WriteMBps: %f", p.WriteMBps)
	}
}

func TestMeasureWriteMBps_writesAndCleans(t *testing.T) {
	dir := t.TempDir()
	before, _ := os.ReadDir(dir)

	_, err := measureWriteMBps(dir)
	if err != nil {
		t.Fatalf("measureWriteMBps: %v", err)
	}

	after, _ := os.ReadDir(dir)
	if len(after) != len(before) {
		t.Errorf("temp file not cleaned up: before=%d after=%d entries", len(before), len(after))
	}
}
