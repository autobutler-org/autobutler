// Package videoutil provides a thin wrapper around ffmpeg and ffprobe for
// video probing, frame extraction, trimming, and transcoding.
//
// All functions return an error if ffmpeg/ffprobe is not available; callers
// should check [Available] at startup and return 501 Not Implemented for
// video-specific endpoints when the tools are missing.
package videoutil

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// TranscodePreset is a named output quality preset for [Transcode].
type TranscodePreset string

const (
	PresetH264720p  TranscodePreset = "h264_720p"
	PresetH2641080p TranscodePreset = "h264_1080p"
	PresetH264480p  TranscodePreset = "h264_480p"
	PresetWebM720p  TranscodePreset = "webm_720p"
)

// Available returns true if both ffmpeg and ffprobe are found on PATH.
func Available() bool {
	_, errMpeg := exec.LookPath("ffmpeg")
	_, errProbe := exec.LookPath("ffprobe")
	return errMpeg == nil && errProbe == nil
}

// Version returns the ffmpeg version string (first line), or an error if not installed.
func Version() (string, error) {
	path, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", fmt.Errorf("ffmpeg not found on PATH: %w", err)
	}
	out, err := exec.Command(path, "-version").Output()
	if err != nil {
		return "", fmt.Errorf("ffmpeg -version: %w", err)
	}
	for _, line := range []byte(out) {
		_ = line
		break
	}
	// Return first line only.
	s := string(out)
	for i, c := range s {
		if c == '\n' {
			return s[:i], nil
		}
	}
	return s, nil
}

// ExtractFrame writes a single JPEG frame at timestamp to outPath.
// outPath should not already exist; use storageutil.GetNonConflictingPath beforehand.
func ExtractFrame(ctx context.Context, filePath string, timestamp time.Duration, outPath string) error {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found: %w", err)
	}
	ts := formatTimestamp(timestamp)
	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-ss", ts,
		"-i", filePath,
		"-frames:v", "1",
		"-q:v", "2",
		"-y",
		outPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg extract frame: %w\n%s", err, out)
	}
	return nil
}

// Trim extracts the clip [startTime, endTime) from filePath and writes it to
// outPath. Uses stream copy when possible (fast, lossless).
func Trim(ctx context.Context, filePath string, startTime, endTime time.Duration, outPath string) error {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found: %w", err)
	}
	duration := endTime - startTime
	if duration <= 0 {
		return fmt.Errorf("endTime must be after startTime")
	}
	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-ss", formatTimestamp(startTime),
		"-i", filePath,
		"-t", formatTimestamp(duration),
		"-c", "copy",
		"-avoid_negative_ts", "make_zero",
		"-y",
		outPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg trim: %w\n%s", err, out)
	}
	return nil
}

// Transcode converts filePath to outPath using the given preset.
func Transcode(ctx context.Context, filePath string, preset TranscodePreset, outPath string) error {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found: %w", err)
	}

	args, err := transcodeArgs(filePath, preset, outPath)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg transcode: %w\n%s", err, out)
	}
	return nil
}

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
