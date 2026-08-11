package ratelimitutil_test

import (
	"testing"

	"github.com/autobutler-org/autobutler/pkg/util/ratelimitutil"
	"golang.org/x/time/rate"
)

func TestAllow_UnderLimit(t *testing.T) {
	l := ratelimitutil.NewWithRate(10, 10)
	for i := range 10 {
		if !l.Allow("1.2.3.4") {
			t.Fatalf("expected Allow on request %d", i)
		}
	}
}

func TestAllow_ExceedsLimit(t *testing.T) {
	// Burst of 2 — 3rd request should be denied.
	l := ratelimitutil.NewWithRate(rate.Limit(0.001), 2)
	if !l.Allow("1.2.3.4") {
		t.Fatal("expected first allow")
	}
	if !l.Allow("1.2.3.4") {
		t.Fatal("expected second allow")
	}
	if l.Allow("1.2.3.4") {
		t.Fatal("expected third to be denied")
	}
}

func TestAllow_DifferentIPsAreIndependent(t *testing.T) {
	// Burst of 1 — first request per IP should pass regardless.
	l := ratelimitutil.NewWithRate(rate.Limit(0.001), 1)
	if !l.Allow("1.1.1.1") {
		t.Fatal("1.1.1.1 first request denied")
	}
	if !l.Allow("2.2.2.2") {
		t.Fatal("2.2.2.2 first request denied")
	}
	// Both exhausted now
	if l.Allow("1.1.1.1") {
		t.Fatal("1.1.1.1 second request should be denied")
	}
}

// TestAllowN_ConsumesMultipleTokens verifies that AllowN deducts n tokens
// from the bucket, so a burst-2 limiter is exhausted after one AllowN(2).
func TestAllowN_ConsumesMultipleTokens(t *testing.T) {
	l := ratelimitutil.NewWithRate(rate.Limit(0.001), 2)
	// Consume both tokens at once.
	if !l.AllowN("1.2.3.4", 2) {
		t.Fatal("expected AllowN(2) to succeed with burst=2")
	}
	// No tokens left.
	if l.AllowN("1.2.3.4", 1) {
		t.Fatal("expected AllowN(1) to fail after burst exhausted")
	}
}

// TestAllowN_PartialConsumption verifies that AllowN(1) is equivalent to
// Allow when n=1.
func TestAllowN_PartialConsumption(t *testing.T) {
	l := ratelimitutil.NewWithRate(rate.Limit(0.001), 3)
	if !l.AllowN("10.0.0.1", 1) {
		t.Fatal("expected first AllowN(1) to succeed")
	}
	if !l.AllowN("10.0.0.1", 1) {
		t.Fatal("expected second AllowN(1) to succeed")
	}
	if !l.AllowN("10.0.0.1", 1) {
		t.Fatal("expected third AllowN(1) to succeed")
	}
	// Burst of 3 exhausted.
	if l.AllowN("10.0.0.1", 1) {
		t.Fatal("expected fourth AllowN(1) to fail")
	}
}

// TestAllowN_ExceedsBurst verifies that requesting more tokens than the burst
// size always fails immediately.
func TestAllowN_ExceedsBurst(t *testing.T) {
	l := ratelimitutil.NewWithRate(rate.Limit(0.001), 2)
	if l.AllowN("9.9.9.9", 3) {
		t.Fatal("expected AllowN(n > burst) to always fail")
	}
}

// TestAllowN_DifferentIPsAreIndependent verifies that AllowN per-IP state is
// isolated — exhausting one IP doesn't affect another.
func TestAllowN_DifferentIPsAreIndependent(t *testing.T) {
	l := ratelimitutil.NewWithRate(rate.Limit(0.001), 1)
	l.AllowN("a.a.a.a", 1) // exhaust a.a.a.a
	if !l.AllowN("b.b.b.b", 1) {
		t.Fatal("b.b.b.b should still have tokens")
	}
}

func TestExtractIP(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"1.2.3.4:8080", "1.2.3.4"},
		{"1.2.3.4", "1.2.3.4"},
		{"::1", "::1"},
		{"[::1]:9090", "::1"},
	}
	for _, tc := range cases {
		got := ratelimitutil.ExtractIP(tc.in)
		if got != tc.want {
			t.Errorf("ExtractIP(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
