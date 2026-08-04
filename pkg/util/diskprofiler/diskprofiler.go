// Package diskprofiler measures storage device IO speed and classifies devices
// into tiers so callers can adapt behaviour (concurrency, caching aggressiveness)
// to the actual underlying hardware.
//
// Profiling runs a short sequential-read benchmark followed by a random-seek
// benchmark against a temporary file in the target directory. Total wall time
// is typically 200–500 ms and leaves no permanent state.
package diskprofiler

import (
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"time"
)

// Class is the speed tier of a storage device.
type Class string

const (
	// ClassFast covers NVMe SSDs and high-speed SATA SSDs (≥150 MB/s sequential).
	ClassFast Class = "fast"
	// ClassMedium covers SATA HDDs and USB 3.x drives with good controllers (≥30 MB/s).
	ClassMedium Class = "medium"
	// ClassSlow covers USB 2.0 HDDs, old mechanical drives, and SD cards (<30 MB/s).
	ClassSlow Class = "slow"
)

const (
	// fastThresholdMBps is the minimum sequential read speed for ClassFast.
	fastThresholdMBps = 150.0
	// mediumThresholdMBps is the minimum sequential read speed for ClassMedium.
	mediumThresholdMBps = 30.0

	// probeSize is the size of the temp file written and read during profiling (4 MB).
	probeSize = 4 * 1024 * 1024
	// randomReads is the number of 4 KB random reads for the latency probe.
	randomReads = 10
	// randomBlockSize is the size of each random read (4 KB).
	randomBlockSize = 4 * 1024
)

// Result contains the profiling outcome for a single storage directory.
type Result struct {
	// SeqReadMBps is the measured sequential read throughput in megabytes per second.
	SeqReadMBps float64
	// RandReadLatencyMs is the average latency of a 4 KB random read in milliseconds.
	RandReadLatencyMs float64
	// Class is the speed tier derived from SeqReadMBps.
	Class Class
	// RecommendedConcurrency is the suggested IOSemaphore slot count for this device.
	RecommendedConcurrency int
}

// Classify returns the Class for a given sequential read speed.
func Classify(seqReadMBps float64) Class {
	switch {
	case seqReadMBps >= fastThresholdMBps:
		return ClassFast
	case seqReadMBps >= mediumThresholdMBps:
		return ClassMedium
	default:
		return ClassSlow
	}
}

// RecommendConcurrency returns the recommended IOSemaphore slot count for a
// given device class. Fast disks can handle more concurrent IO; slow disks
// need lower concurrency to avoid seek-thrashing.
func RecommendConcurrency(c Class) int {
	switch c {
	case ClassFast:
		return 16
	case ClassMedium:
		return 8
	default: // ClassSlow
		return 4
	}
}

// Profile measures the IO speed of the storage volume containing dir. It writes
// a temporary file, reads it back sequentially, performs random reads, then
// removes the file. The caller receives a Result with measured throughput, latency,
// and a pre-computed device Class.
//
// Profile is safe to call concurrently on different directories. It will not
// be accurate if the device is already under heavy IO load — for startup
// profiling this is generally acceptable.
func Profile(dir string) (Result, error) {
	f, err := os.CreateTemp(dir, ".diskprobe-*")
	if err != nil {
		return Result{}, fmt.Errorf("diskprofiler: create temp file in %s: %w", dir, err)
	}
	path := f.Name()
	defer os.Remove(path)

	// Write probeSize bytes so we have data to read.
	buf := make([]byte, probeSize)
	// Fill with pseudo-random data so the OS can't optimise away the write.
	for i := range buf {
		buf[i] = byte(rand.IntN(256)) //nolint:gosec // non-crypto; content irrelevant
	}
	if _, err := f.Write(buf); err != nil {
		f.Close()
		return Result{}, fmt.Errorf("diskprofiler: write probe: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return Result{}, fmt.Errorf("diskprofiler: sync probe: %w", err)
	}
	f.Close()

	// Sequential read benchmark.
	seqMBps, err := sequentialRead(path, probeSize)
	if err != nil {
		return Result{}, err
	}

	// Random read latency benchmark.
	randLatMs, err := randomReadLatency(path, probeSize, randomReads, randomBlockSize)
	if err != nil {
		return Result{}, err
	}

	cls := Classify(seqMBps)
	return Result{
		SeqReadMBps:            seqMBps,
		RandReadLatencyMs:      randLatMs,
		Class:                  cls,
		RecommendedConcurrency: RecommendConcurrency(cls),
	}, nil
}

// sequentialRead opens path, reads it fully, and returns throughput in MB/s.
func sequentialRead(path string, size int) (float64, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return 0, fmt.Errorf("diskprofiler: open for seq read: %w", err)
	}
	defer f.Close()

	buf := make([]byte, 256*1024) // 256 KB read buffer
	start := time.Now()
	if _, err := io.CopyBuffer(io.Discard, f, buf); err != nil {
		return 0, fmt.Errorf("diskprofiler: seq read: %w", err)
	}
	elapsed := time.Since(start)
	if elapsed == 0 {
		elapsed = time.Microsecond
	}
	return float64(size) / elapsed.Seconds() / (1024 * 1024), nil
}

// randomReadLatency performs n random 4 KB reads across the file at path and
// returns the average latency per read in milliseconds.
func randomReadLatency(path string, fileSize, n, blockSize int) (float64, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return 0, fmt.Errorf("diskprofiler: open for rand read: %w", err)
	}
	defer f.Close()

	buf := make([]byte, blockSize)
	maxOffset := int64(fileSize - blockSize)
	if maxOffset <= 0 {
		maxOffset = 1
	}

	var totalNs int64
	for i := 0; i < n; i++ {
		offset := rand.Int64N(maxOffset) //nolint:gosec // non-crypto; just a seek offset
		start := time.Now()
		if _, err := f.ReadAt(buf, offset); err != nil && err != io.EOF {
			return 0, fmt.Errorf("diskprofiler: rand read at %d: %w", offset, err)
		}
		totalNs += time.Since(start).Nanoseconds()
	}

	avgMs := float64(totalNs) / float64(n) / float64(time.Millisecond)
	return avgMs, nil
}

// RecommendCleanupTimeout returns a reasonable timeout for background
// filesystem cleanup tasks (moves, async DB writes) based on device class.
// SD cards and USB 2.0 drives need longer timeouts than NVMe SSDs.
func RecommendCleanupTimeout(c Class) time.Duration {
	switch c {
	case ClassFast:
		return 10 * time.Second
	case ClassMedium:
		return 30 * time.Second
	default: // ClassSlow (SD cards, USB 2.0)
		return 90 * time.Second
	}
}
