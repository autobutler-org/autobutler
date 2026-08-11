// Package diskprofiler classifies a storage path into one of three speed tiers
// (Fast, Medium, Slow) by running a short sequential write benchmark.
//
// The primary use-case is the Raspberry Pi, where the storage device can be an
// NVMe SSD, a USB 3 drive, or a slow SD card — each with wildly different IO
// throughput. Callers use the tier to tune concurrency limits and timeouts so
// that SD cards are not thrashed and SSDs can run at full speed.
//
// Usage:
//
//	tier, err := diskprofiler.Profile(ctx, "/path/to/data")
//	timeout := tier.DeleteTimeout()     // 5s / 30s / 60s
//	workers := tier.UploadWorkers()     // 8 / 4 / 2
package diskprofiler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Tier classifies storage device speed.
type Tier int

const (
	// TierFast covers NVMe / SATA SSD and fast USB 3 drives (≥ 50 MB/s sequential write).
	TierFast Tier = iota
	// TierMedium covers USB 2/3 HDDs and mid-range flash storage (10–50 MB/s).
	TierMedium
	// TierSlow covers SD cards and slow USB 2 drives (< 10 MB/s).
	TierSlow
)

// String returns a human-readable label for the tier.
func (t Tier) String() string {
	switch t {
	case TierFast:
		return "fast"
	case TierMedium:
		return "medium"
	case TierSlow:
		return "slow"
	default:
		return "unknown"
	}
}

// DeleteTimeout returns the recommended timeout for an async delete operation
// on a device of this tier.
//   - Fast:   5 s  (SSD — deletes are near-instant)
//   - Medium: 30 s (HDD / USB — modest IO queuing expected)
//   - Slow:   60 s (SD card — give it time, don't thrash)
func (t Tier) DeleteTimeout() time.Duration {
	switch t {
	case TierFast:
		return 5 * time.Second
	case TierMedium:
		return 30 * time.Second
	default: // TierSlow
		return 60 * time.Second
	}
}

// UploadWorkers returns the recommended maximum number of concurrent upload
// goroutines for a device of this tier.
//   - Fast:   8 (SSD handles parallelism well)
//   - Medium: 4 (HDD benefits from some parallelism)
//   - Slow:   2 (SD card — sequential is safer)
func (t Tier) UploadWorkers() int {
	switch t {
	case TierFast:
		return 8
	case TierMedium:
		return 4
	default: // TierSlow
		return 2
	}
}

// SessionPurgeBatchSize returns the recommended number of sessions to delete
// per batch during the periodic session purge.
//   - Fast:   500
//   - Medium: 100
//   - Slow:    20
func (t Tier) SessionPurgeBatchSize() int {
	switch t {
	case TierFast:
		return 500
	case TierMedium:
		return 100
	default: // TierSlow
		return 20
	}
}

// benchmarkSize is the amount of data written during a profile run.
// 4 MB is large enough to exceed the OS page cache on most systems while
// remaining fast enough that even a slow SD card finishes in under 5 seconds.
const benchmarkSize = 4 * 1024 * 1024 // 4 MB

// fastThresholdBytesPerSec is the lower bound for TierFast (50 MB/s).
const fastThresholdBytesPerSec = 50 * 1024 * 1024

// mediumThresholdBytesPerSec is the lower bound for TierMedium (10 MB/s).
const mediumThresholdBytesPerSec = 10 * 1024 * 1024

// Profile runs a short sequential write benchmark in dir and returns the Tier.
// The benchmark writes and immediately removes a temporary file — it does not
// persist any data. dir must be writable.
//
// If the benchmark cannot complete within ctx's deadline, or if dir is not
// writable, Profile returns (TierSlow, err) so callers can safely fall back to
// the most conservative settings.
func Profile(ctx context.Context, dir string) (Tier, error) {
	tmp, err := os.CreateTemp(dir, ".diskprofiler-*")
	if err != nil {
		return TierSlow, fmt.Errorf("diskprofiler: create temp file in %q: %w", dir, err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	buf := make([]byte, 64*1024) // 64 KB write buffer
	for i := range buf {
		buf[i] = byte(i & 0xFF)
	}

	written := 0
	start := time.Now()

	for written < benchmarkSize {
		select {
		case <-ctx.Done():
			return TierSlow, ctx.Err()
		default:
		}
		n, err := tmp.Write(buf)
		if err != nil {
			return TierSlow, fmt.Errorf("diskprofiler: write error: %w", err)
		}
		written += n
	}

	// Sync to flush OS write cache to the block device.
	if err := tmp.Sync(); err != nil {
		return TierSlow, fmt.Errorf("diskprofiler: sync error: %w", err)
	}

	elapsed := time.Since(start)
	if elapsed == 0 {
		elapsed = time.Nanosecond // avoid divide-by-zero on pathological clocks
	}

	bytesPerSec := float64(written) / elapsed.Seconds()
	return classify(bytesPerSec), nil
}

// ProfilePath is a convenience wrapper that resolves the directory from a file
// path (uses filepath.Dir) and delegates to Profile.
func ProfilePath(ctx context.Context, path string) (Tier, error) {
	return Profile(ctx, filepath.Dir(path))
}

// classify converts a measured throughput to a Tier.
func classify(bytesPerSec float64) Tier {
	switch {
	case bytesPerSec >= fastThresholdBytesPerSec:
		return TierFast
	case bytesPerSec >= mediumThresholdBytesPerSec:
		return TierMedium
	default:
		return TierSlow
	}
}
