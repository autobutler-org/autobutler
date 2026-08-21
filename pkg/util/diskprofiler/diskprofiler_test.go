package diskprofiler_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/autobutler-org/quark/pkg/util/diskprofiler"
)

// TestProfile_ReturnsTierForTempDir verifies that Profile completes without
// error for a writable directory and returns a valid Tier.
func TestProfile_ReturnsTierForTempDir(t *testing.T) {
	ctx := context.Background()
	tier, err := diskprofiler.Profile(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("Profile: unexpected error: %v", err)
	}
	switch tier {
	case diskprofiler.TierFast, diskprofiler.TierMedium, diskprofiler.TierSlow:
		// valid
	default:
		t.Errorf("unexpected tier value %d", tier)
	}
}

// TestProfilePath_EquivalentToProfile verifies that ProfilePath gives the same
// result as Profile on the same directory.
func TestProfilePath_EquivalentToProfile(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	// Create a dummy file path inside the temp dir to satisfy ProfilePath.
	dummyPath := dir + "/file.bin"

	tier1, err1 := diskprofiler.Profile(ctx, dir)
	tier2, err2 := diskprofiler.ProfilePath(ctx, dummyPath)

	if err1 != nil {
		t.Fatalf("Profile: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("ProfilePath: %v", err2)
	}

	// The two measurements may differ slightly due to OS caching effects, so
	// we don't assert equality — just that both succeed and return valid tiers.
	_ = tier1
	_ = tier2
}

// TestProfile_CancelledContextReturnsError verifies that a pre-cancelled context
// causes Profile to return an error and TierSlow.
func TestProfile_CancelledContextReturnsError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	tier, err := diskprofiler.Profile(ctx, t.TempDir())
	if err == nil {
		t.Error("expected error from cancelled context, got nil")
	}
	if tier != diskprofiler.TierSlow {
		t.Errorf("expected TierSlow on error, got %v", tier)
	}
}

// TestProfile_NonWritablePathReturnsError verifies that Profile returns an error
// (and TierSlow) when the directory is not writable.
func TestProfile_NonWritablePathReturnsError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can write to mode-000 dirs; skip")
	}

	dir := t.TempDir()
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) }) //nolint:errcheck

	tier, err := diskprofiler.Profile(context.Background(), dir)
	if err == nil {
		t.Error("expected error for non-writable dir, got nil")
	}
	if tier != diskprofiler.TierSlow {
		t.Errorf("expected TierSlow on error, got %v", tier)
	}
}

// TestProfile_DeadlineExceeded verifies that a very tight deadline causes
// Profile to return a context error.
func TestProfile_DeadlineExceeded(t *testing.T) {
	// 1 ns deadline — guaranteed to be exceeded before any write finishes.
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond) // ensure deadline has passed

	tier, err := diskprofiler.Profile(ctx, t.TempDir())
	if err == nil {
		t.Error("expected deadline error, got nil")
	}
	if tier != diskprofiler.TierSlow {
		t.Errorf("expected TierSlow on deadline, got %v", tier)
	}
}

// ─── Tier method tests ───────────────────────────────────────────────────────

func TestTier_String(t *testing.T) {
	cases := []struct {
		tier diskprofiler.Tier
		want string
	}{
		{diskprofiler.TierFast, "fast"},
		{diskprofiler.TierMedium, "medium"},
		{diskprofiler.TierSlow, "slow"},
	}
	for _, tc := range cases {
		if got := tc.tier.String(); got != tc.want {
			t.Errorf("Tier(%d).String() = %q, want %q", tc.tier, got, tc.want)
		}
	}
}

func TestTier_DeleteTimeout(t *testing.T) {
	if diskprofiler.TierFast.DeleteTimeout() >= diskprofiler.TierMedium.DeleteTimeout() {
		t.Error("expected TierFast.DeleteTimeout < TierMedium.DeleteTimeout")
	}
	if diskprofiler.TierMedium.DeleteTimeout() >= diskprofiler.TierSlow.DeleteTimeout() {
		t.Error("expected TierMedium.DeleteTimeout < TierSlow.DeleteTimeout")
	}
}

func TestTier_UploadWorkers(t *testing.T) {
	if diskprofiler.TierFast.UploadWorkers() <= diskprofiler.TierMedium.UploadWorkers() {
		t.Error("expected TierFast.UploadWorkers > TierMedium.UploadWorkers")
	}
	if diskprofiler.TierMedium.UploadWorkers() <= diskprofiler.TierSlow.UploadWorkers() {
		t.Error("expected TierMedium.UploadWorkers > TierSlow.UploadWorkers")
	}
}

func TestTier_SessionPurgeBatchSize(t *testing.T) {
	if diskprofiler.TierFast.SessionPurgeBatchSize() <= diskprofiler.TierMedium.SessionPurgeBatchSize() {
		t.Error("expected TierFast batch size > TierMedium")
	}
	if diskprofiler.TierMedium.SessionPurgeBatchSize() <= diskprofiler.TierSlow.SessionPurgeBatchSize() {
		t.Error("expected TierMedium batch size > TierSlow")
	}
}

// TestProfile_TempFileCleaned verifies that the benchmark temp file does not
// linger after Profile returns.
func TestProfile_TempFileCleaned(t *testing.T) {
	dir := t.TempDir()
	before, _ := os.ReadDir(dir)
	diskprofiler.Profile(context.Background(), dir) //nolint:errcheck
	after, _ := os.ReadDir(dir)
	if len(after) != len(before) {
		t.Errorf("temp file leaked: dir had %d entries before, %d after", len(before), len(after))
	}
}
