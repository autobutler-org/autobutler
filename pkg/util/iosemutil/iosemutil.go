// Package iosemutil provides a shared IO concurrency semaphore to prevent disk
// thrashing under concurrent load — particularly on spinning hard drives where
// random-access seeks collapse throughput when too many goroutines compete.
//
// Usage: obtain the semaphore before any disk-bound operation and release it
// when done. Operations that cannot acquire the semaphore within the timeout
// should return HTTP 503 with a Retry-After header rather than blocking
// indefinitely.
package iosemutil

import (
	"context"
	"time"
)

const (
	// DefaultConcurrency is the default number of concurrent IO operations
	// allowed. Tuned for spinning HDDs — lower this if disk contention is
	// observed; SSD deployments can raise it.
	DefaultConcurrency = 8

	// DefaultTimeout is the maximum time a caller will wait to acquire the
	// semaphore before giving up. Callers should return HTTP 503 on timeout.
	DefaultTimeout = 30 * time.Second
)

// Semaphore is a counting semaphore that limits concurrent disk-bound
// operations. The zero value is not valid — use New or NewWithConcurrency.
type Semaphore struct {
	ch chan struct{}
}

// New returns a Semaphore with DefaultConcurrency slots.
func New() *Semaphore {
	return NewWithConcurrency(DefaultConcurrency)
}

// NewWithConcurrency returns a Semaphore limited to n concurrent holders.
// Panics if n < 1.
func NewWithConcurrency(n int) *Semaphore {
	if n < 1 {
		panic("iosemutil: concurrency must be >= 1")
	}
	return &Semaphore{ch: make(chan struct{}, n)}
}

// Acquire waits up to timeout to obtain one slot from the semaphore.
// Returns true if the slot was acquired; false if the context or timeout
// expired. Callers must call Release exactly once if Acquire returns true.
func (s *Semaphore) Acquire(ctx context.Context, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	select {
	case s.ch <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

// AcquireDefault calls Acquire with DefaultTimeout.
func (s *Semaphore) AcquireDefault(ctx context.Context) bool {
	return s.Acquire(ctx, DefaultTimeout)
}

// Release returns a slot to the semaphore. Must be called exactly once per
// successful Acquire, typically via defer.
func (s *Semaphore) Release() {
	<-s.ch
}

// Available returns the number of free slots (for monitoring/logging).
func (s *Semaphore) Available() int {
	return cap(s.ch) - len(s.ch)
}

// Cap returns the maximum concurrency of this semaphore.
func (s *Semaphore) Cap() int {
	return cap(s.ch)
}
