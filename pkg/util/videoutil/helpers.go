package videoutil

import (
	"fmt"
	"time"
)

// transcodeArgs builds the ffmpeg argument list for the given preset.
func transcodeArgs(filePath string, preset TranscodePreset, outPath string) ([]string, error) {
	base := []string{"-i", filePath}
	var encArgs []string

	switch preset {
	case PresetH264480p:
		encArgs = []string{"-vf", "scale=-2:480", "-c:v", "libx264", "-crf", "23", "-preset", "fast", "-c:a", "aac", "-b:a", "128k"}
	case PresetH264720p:
		encArgs = []string{"-vf", "scale=-2:720", "-c:v", "libx264", "-crf", "23", "-preset", "fast", "-c:a", "aac", "-b:a", "128k"}
	case PresetH2641080p:
		encArgs = []string{"-vf", "scale=-2:1080", "-c:v", "libx264", "-crf", "23", "-preset", "fast", "-c:a", "aac", "-b:a", "192k"}
	case PresetWebM720p:
		encArgs = []string{"-vf", "scale=-2:720", "-c:v", "libvpx-vp9", "-crf", "33", "-b:v", "0", "-c:a", "libopus", "-b:a", "128k"}
	default:
		return nil, fmt.Errorf("unknown transcode preset: %q", preset)
	}

	return append(append(base, encArgs...), "-y", outPath), nil
}

// formatTimestamp converts a Duration to an ffmpeg timestamp string (HH:MM:SS.mmm).
func formatTimestamp(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	ms := int(d.Milliseconds()) % 1000
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}
