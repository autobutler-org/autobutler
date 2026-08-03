package videoutil

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// VideoInfo holds the metadata returned by ffprobe for a single video file.
type VideoInfo struct {
	Duration   time.Duration
	Width      int
	Height     int
	VideoCodec string
	AudioCodec string
	Bitrate    int64
	Framerate  float64
	Rotation   int // degrees (0, 90, 180, 270)
}

// ffprobeOutput is the top-level JSON structure returned by ffprobe -print_format json.
type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

type ffprobeStream struct {
	CodecType          string            `json:"codec_type"`
	CodecName          string            `json:"codec_name"`
	Width              int               `json:"width"`
	Height             int               `json:"height"`
	RFrameRate         string            `json:"r_frame_rate"` // e.g. "30000/1001"
	AvgFrameRate       string            `json:"avg_frame_rate"`
	SideDataList       []ffprobeSideData `json:"side_data_list"`
	Tags               map[string]string `json:"tags"`
}

type ffprobeSideData struct {
	SideDataType string `json:"side_data_type"`
	Rotation     int    `json:"rotation"`
}

type ffprobeFormat struct {
	Duration string `json:"duration"` // seconds as string, e.g. "127.400000"
	BitRate  string `json:"bit_rate"` // bits/s as string
}

// Probe runs ffprobe on filePath and returns a VideoInfo.
// Returns an error if ffprobe is not available or the file is not a valid video.
func Probe(ctx context.Context, filePath string) (*VideoInfo, error) {
	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		return nil, fmt.Errorf("ffprobe not found on PATH: %w", err)
	}

	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	}
	cmd := exec.CommandContext(ctx, ffprobePath, args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed for %q: %w", filePath, err)
	}

	var probe ffprobeOutput
	if err := json.Unmarshal(out, &probe); err != nil {
		return nil, fmt.Errorf("parse ffprobe output: %w", err)
	}

	info := &VideoInfo{}

	// Format-level fields.
	if probe.Format.Duration != "" {
		if secs, err := strconv.ParseFloat(strings.TrimSpace(probe.Format.Duration), 64); err == nil {
			info.Duration = time.Duration(secs * float64(time.Second))
		}
	}
	if probe.Format.BitRate != "" {
		if br, err := strconv.ParseInt(strings.TrimSpace(probe.Format.BitRate), 10, 64); err == nil {
			info.Bitrate = br
		}
	}

	// Stream-level fields — first video stream and first audio stream.
	for _, s := range probe.Streams {
		switch s.CodecType {
		case "video":
			if info.VideoCodec == "" {
				info.VideoCodec = s.CodecName
				info.Width = s.Width
				info.Height = s.Height
				info.Framerate = parseFramerate(s.AvgFrameRate)
				if info.Framerate == 0 {
					info.Framerate = parseFramerate(s.RFrameRate)
				}
				// Rotation may be in side-data or in stream tags.
				info.Rotation = extractRotation(s)
			}
		case "audio":
			if info.AudioCodec == "" {
				info.AudioCodec = s.CodecName
			}
		}
	}

	return info, nil
}

// parseFramerate converts an ffprobe fraction string ("30000/1001", "30/1") to float64.
func parseFramerate(s string) float64 {
	parts := strings.SplitN(strings.TrimSpace(s), "/", 2)
	if len(parts) != 2 {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			return v
		}
		return 0
	}
	num, err1 := strconv.ParseFloat(parts[0], 64)
	den, err2 := strconv.ParseFloat(parts[1], 64)
	if err1 != nil || err2 != nil || den == 0 {
		return 0
	}
	return num / den
}

// extractRotation reads the display rotation from side-data or tags.
// Returns 0 if no rotation is found.
func extractRotation(s ffprobeStream) int {
	// Prefer structured side-data (newer ffprobe).
	for _, sd := range s.SideDataList {
		if strings.Contains(strings.ToLower(sd.SideDataType), "display matrix") {
			// Rotation is stored as a negative angle in ffprobe side-data.
			r := sd.Rotation
			if r < 0 {
				r += 360
			}
			return r
		}
	}
	// Fall back to stream tags (older ffprobe / some containers).
	if tag, ok := s.Tags["rotate"]; ok {
		if r, err := strconv.Atoi(strings.TrimSpace(tag)); err == nil {
			if r < 0 {
				r += 360
			}
			return r
		}
	}
	return 0
}
