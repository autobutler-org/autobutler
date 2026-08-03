package videoutil

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestAvailable checks that Available() reflects whether ffmpeg/ffprobe are on PATH.
func TestAvailable(t *testing.T) {
	// Just verifies the function doesn't panic.
	_ = Available()
}

// TestVersion checks that Version() returns a non-empty string when ffmpeg is available.
func TestVersion(t *testing.T) {
	if !Available() {
		t.Skip("ffmpeg not available")
	}
	v, err := Version()
	if err != nil {
		t.Fatalf("Version() error: %v", err)
	}
	if v == "" {
		t.Fatal("Version() returned empty string")
	}
}

// TestProbeRealFile probes a real video file if VIDEOUTIL_TEST_FILE is set.
func TestProbeRealFile(t *testing.T) {
	if !Available() {
		t.Skip("ffprobe not available")
	}
	filePath := os.Getenv("VIDEOUTIL_TEST_FILE")
	if filePath == "" {
		t.Skip("set VIDEOUTIL_TEST_FILE to a real video path to run this test")
	}
	info, err := Probe(context.Background(), filePath)
	if err != nil {
		t.Fatalf("Probe() error: %v", err)
	}
	if info.Duration <= 0 {
		t.Errorf("expected positive Duration, got %v", info.Duration)
	}
	if info.Width <= 0 || info.Height <= 0 {
		t.Errorf("expected positive dimensions, got %dx%d", info.Width, info.Height)
	}
	t.Logf("Probe result: %+v", info)
}

// TestFormatTimestamp exercises the timestamp formatter used by ExtractFrame and Trim.
func TestFormatTimestamp(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "00:00:00.000"},
		{time.Second, "00:00:01.000"},
		{90*time.Second + 500*time.Millisecond, "00:01:30.500"},
		{3661*time.Second + 123*time.Millisecond, "01:01:01.123"},
	}
	for _, c := range cases {
		got := formatTimestamp(c.d)
		if got != c.want {
			t.Errorf("formatTimestamp(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}
