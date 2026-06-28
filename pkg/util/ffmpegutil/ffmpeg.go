package ffmpegutil

import (
	"context"
	"time"
)

type VideoKind string

const (
	VideoKindMP4  VideoKind = "mp4"
	VideoKindMOV  VideoKind = "mov"
	VideoKindMKV  VideoKind = "mkv"
	VideoKindAVI  VideoKind = "avi"
	VideoKindWebM VideoKind = "webm"
)

type MediaInfo struct {
	Duration time.Duration
	Width    int
	Height   int
	Codec    string
	Format   string
	Bitrate  int64
}

type VideoProcessor interface {
	Run(ctx context.Context, args ...string) error
	Probe(ctx context.Context, path string) (*MediaInfo, error)
	Convert(ctx context.Context, src, dst string, from, to VideoKind) error
	ExtractThumbnail(ctx context.Context, src, dst string, at time.Duration) error
	Trim(ctx context.Context, src, dst string, start, end time.Duration) error
	Available() bool
}
