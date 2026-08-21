package iosemutil_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/autobutler-org/quark/pkg/util/iosemutil"
)

func TestNew(t *testing.T) {
	s := iosemutil.New()
	if s == nil {
		t.Fatal("New() returned nil")
	}
	if got := s.Cap(); got != iosemutil.DefaultConcurrency {
		t.Errorf("Cap() = %d, want %d", got, iosemutil.DefaultConcurrency)
	}
	if got := s.Available(); got != iosemutil.DefaultConcurrency {
		t.Errorf("Available() = %d before any acquires, want %d", got, iosemutil.DefaultConcurrency)
	}
}

func TestNewWithConcurrency(t *testing.T) {
	s := iosemutil.NewWithConcurrency(3)
	if s.Cap() != 3 {
		t.Errorf("Cap() = %d, want 3", s.Cap())
	}
}

func TestNewWithConcurrencyPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for concurrency < 1")
		}
	}()
	iosemutil.NewWithConcurrency(0)
}

func TestAcquireAndRelease(t *testing.T) {
	s := iosemutil.NewWithConcurrency(2)
	ctx := context.Background()

	if !s.Acquire(ctx, time.Second) {
		t.Fatal("first Acquire should succeed")
	}
	if s.Available() != 1 {
		t.Errorf("Available() = %d after one acquire, want 1", s.Available())
	}

	s.Release()
	if s.Available() != 2 {
		t.Errorf("Available() = %d after release, want 2", s.Available())
	}
}

func TestAcquireDefault(t *testing.T) {
	s := iosemutil.NewWithConcurrency(1)
	ctx := context.Background()
	if !s.AcquireDefault(ctx) {
		t.Fatal("AcquireDefault should succeed on empty semaphore")
	}
	s.Release()
}

func TestAcquireTimeout(t *testing.T) {
	s := iosemutil.NewWithConcurrency(1)
	ctx := context.Background()

	if !s.Acquire(ctx, time.Second) {
		t.Fatal("first Acquire should succeed")
	}
	// Second acquire should time out quickly
	if s.Acquire(ctx, 10*time.Millisecond) {
		t.Error("second Acquire should have timed out")
	}
	s.Release()
}

func TestAcquireContextCancelled(t *testing.T) {
	s := iosemutil.NewWithConcurrency(1)
	ctx, cancel := context.WithCancel(context.Background())

	if !s.Acquire(ctx, time.Second) {
		t.Fatal("first Acquire should succeed")
	}

	cancel() // cancel the context
	if s.Acquire(ctx, time.Second) {
		t.Error("Acquire with cancelled context should fail")
	}
	s.Release()
}

func TestConcurrentLimit(t *testing.T) {
	const limit = 3
	s := iosemutil.NewWithConcurrency(limit)
	ctx := context.Background()

	var mu sync.Mutex
	maxConcurrent := 0
	current := 0

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !s.Acquire(ctx, 5*time.Second) {
				t.Errorf("Acquire timed out unexpectedly")
				return
			}
			defer s.Release()

			mu.Lock()
			current++
			if current > maxConcurrent {
				maxConcurrent = current
			}
			if current > limit {
				t.Errorf("concurrent holders = %d, exceeds limit %d", current, limit)
			}
			mu.Unlock()

			time.Sleep(5 * time.Millisecond) // simulate IO

			mu.Lock()
			current--
			mu.Unlock()
		}()
	}
	wg.Wait()

	if maxConcurrent > limit {
		t.Errorf("peak concurrency %d exceeded limit %d", maxConcurrent, limit)
	}
}
