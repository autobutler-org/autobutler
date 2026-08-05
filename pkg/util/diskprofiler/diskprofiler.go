// Package diskprofiler classifies storage devices into speed tiers based on
// a short sequential-write benchmark. The tier drives IO-adaptive behaviour
// (concurrency limits, timeouts) so the butler performs well on both fast
// SSDs and slow SD cards without manual configuration.
package diskprofiler

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Tier represents the measured IO speed class of a storage device.
type Tier int

const (
	// TierFast covers NVMe / SATA SSD — sequential write > 50 MB/s.
	// Suitable for high concurrency and short timeouts.
	TierFast Tier = iota
	// TierMedium covers USB 3.0 HDD or fast SD card (A2) — 10–50 MB/s.
	TierMedium
	// TierSlow covers SD card (A1/Class 10) or USB 2.0 HDD — < 10 MB/s.
	// Requires low concurrency and generous timeouts to avoid thrashing.
	TierSlow
)

// String returns a human-readable tier name.
func (t Tier) String() string {
	switch t {
	case TierFast:
		return "fast"
	case TierMedium:
		return "medium"
	default:
		return "slow"
	}
}

// Profile holds the benchmark result for a device.
type Profile struct {
	// Dir is the path that was benchmarked.
	Dir string
	// Tier is the classified speed class.
	Tier Tier
	// WriteMBps is the measured sequential-write throughput in MB/s.
	WriteMBps float64
	// MaxConcurrency is the recommended number of concurrent IO operations.
	MaxConcurrency int
	// CleanupTimeout is the recommended timeout for background cleanup jobs.
	CleanupTimeout time.Duration
}

const (
	// benchmarkSize is the number of bytes written in the benchmark.
	// 4 MB is large enough to bypass most OS write buffers without taking
	// more than a few seconds even on slow media.
	benchmarkSize = 4 * 1024 * 1024

	// fastThresholdMBps — above this → TierFast.
	fastThresholdMBps = 50.0
	// mediumThresholdMBps — above this → TierMedium (else TierSlow).
	mediumThresholdMBps = 10.0
)

// Measure runs a short sequential-write benchmark on dir and returns the
// classified Profile. The benchmark writes a 4 MB temporary file with
// O_SYNC to bypass the page cache, then deletes it.
//
// If the benchmark fails (e.g. read-only filesystem), the function returns
// TierSlow conservatively — callers should treat unknown devices as slow.
func Measure(dir string) Profile {
	p := Profile{Dir: dir}

	mbps, err := measureWriteMBps(dir)
	if err != nil {
		// Conservative fallback: treat as slow so we don't thrash.
		p.Tier = TierSlow
		p.WriteMBps = 0
		p.MaxConcurrency = tierParams(TierSlow)
		p.CleanupTimeout = tierTimeout(TierSlow)
		return p
	}

	p.WriteMBps = mbps
	switch {
	case mbps >= fastThresholdMBps:
		p.Tier = TierFast
	case mbps >= mediumThresholdMBps:
		p.Tier = TierMedium
	default:
		p.Tier = TierSlow
	}
	p.MaxConcurrency = tierParams(p.Tier)
	p.CleanupTimeout = tierTimeout(p.Tier)
	return p
}

// MeasureRoot is a convenience wrapper that benchmarks the root filesystem ("/").
func MeasureRoot() Profile {
	return Measure("/")
}

// tierParams returns the recommended concurrent IO goroutines for a tier.
func tierParams(t Tier) int {
	switch t {
	case TierFast:
		return 8
	case TierMedium:
		return 4
	default: // TierSlow
		return 1
	}
}

// tierTimeout returns the recommended cleanup-job timeout for a tier.
func tierTimeout(t Tier) time.Duration {
	switch t {
	case TierFast:
		return 5 * time.Second
	case TierMedium:
		return 15 * time.Second
	default: // TierSlow — SD card write can be very slow under load
		return 60 * time.Second
	}
}

// measureWriteMBps writes benchmarkSize bytes to a temp file in dir with
// O_SYNC and measures the elapsed time. Returns MB/s.
func measureWriteMBps(dir string) (float64, error) {
	f, err := os.CreateTemp(dir, ".diskprofiler-bench-*")
	if err != nil {
		return 0, fmt.Errorf("create temp file: %w", err)
	}
	name := f.Name()
	defer os.Remove(name)
	defer f.Close()

	// Re-open with O_SYNC to bypass the OS write buffer so the measurement
	// reflects actual device throughput rather than RAM speed.
	f.Close()
	f, err = os.OpenFile(name, os.O_WRONLY|os.O_SYNC, 0o600)
	if err != nil {
		return 0, fmt.Errorf("open with O_SYNC: %w", err)
	}
	defer f.Close()

	buf := make([]byte, benchmarkSize)
	// Fill buffer with a non-zero pattern so compressing storage controllers
	// can't shortcut the write.
	for i := range buf {
		buf[i] = byte(i & 0xff)
	}

	start := time.Now()
	if _, err := f.Write(buf); err != nil {
		return 0, fmt.Errorf("write: %w", err)
	}
	elapsed := time.Since(start)
	if elapsed == 0 {
		return 0, fmt.Errorf("elapsed time is zero")
	}

	mbps := float64(benchmarkSize) / elapsed.Seconds() / (1024 * 1024)
	return mbps, nil
}

// CachedProfiler holds a lazily-initialised profile for a directory so the
// benchmark only runs once per process lifetime.
type CachedProfiler struct {
	dir     string
	profile *Profile
}

// NewCachedProfiler creates a CachedProfiler for the given directory.
// Use NewCachedRootProfiler for the root filesystem.
func NewCachedProfiler(dir string) *CachedProfiler {
	return &CachedProfiler{dir: dir}
}

// NewCachedRootProfiler creates a CachedProfiler for "/".
func NewCachedRootProfiler() *CachedProfiler {
	return NewCachedProfiler(filepath.FromSlash("/"))
}

// Get returns the cached Profile, running the benchmark on first call.
func (c *CachedProfiler) Get() Profile {
	if c.profile != nil {
		return *c.profile
	}
	p := Measure(c.dir)
	c.profile = &p
	return p
}

// Reset clears the cached result so the next Get() re-runs the benchmark.
// Useful in tests.
func (c *CachedProfiler) Reset() {
	c.profile = nil
}
