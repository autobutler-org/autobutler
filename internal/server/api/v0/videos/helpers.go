package v0_videos

import (
	"fmt"
	"math"
	"time"
)

// formatFrameTimestamp converts a Duration to a human-readable frame label,
// e.g. 2500ms → "0m02s", 3725000ms → "1h02m05s".
func formatFrameTimestamp(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	}
	return fmt.Sprintf("%dm%02ds", m, s)
}

func roundTo(v float64, decimals int) float64 {
	factor := math.Pow(10, float64(decimals))
	return math.Round(v*factor) / factor
}
