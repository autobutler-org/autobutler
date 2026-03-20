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
