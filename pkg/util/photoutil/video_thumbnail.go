package photoutil

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"time"

	"github.com/autobutler-org/autobutler/pkg/util/ffmpegutil"

	"github.com/KononK/resize"
)

var defaultProcessor ffmpegutil.VideoProcessor = ffmpegutil.NewCLIProcessor()

// VideoToThumbnail extracts a frame from a video file and returns a thumbnail.
// Requires ffmpeg to be installed on the system.
func VideoToThumbnail(filePath string, width, height uint) (image.Image, error) {
	tmpDir, err := os.MkdirTemp("", "autobutler-vidthumb-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	framePath := filepath.Join(tmpDir, "frame.jpg")

	ctx := context.Background()
	err = defaultProcessor.ExtractThumbnail(ctx, filePath, framePath, 1*time.Second)
	if err != nil {
		// Retry at 0s in case the video is shorter than 1 second.
		if err2 := defaultProcessor.ExtractThumbnail(ctx, filePath, framePath, 0); err2 != nil {
			return nil, fmt.Errorf("ffmpeg frame extraction failed: %w", err2)
		}
	}

	f, err := os.Open(framePath)
	if err != nil {
		return nil, fmt.Errorf("open extracted frame: %w", err)
	}
	defer f.Close()

	img, err := jpeg.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode extracted frame: %w", err)
	}

	thumbnail, _, err := cropToFit(img, width, height)
	if err != nil {
		return nil, fmt.Errorf("crop video thumbnail: %w", err)
	}

	return thumbnail, nil
}

// IsFFmpegAvailable checks whether ffmpeg is on the PATH.
func IsFFmpegAvailable() bool {
	return defaultProcessor.Available()
}

// cropToFit scales and center-crops an image to the target dimensions.
func cropToFit(img image.Image, width, height uint) (image.Image, string, error) {
	bounds := img.Bounds()
	srcW := uint(bounds.Dx())
	srcH := uint(bounds.Dy())

	if srcW == 0 || srcH == 0 {
		return nil, "", fmt.Errorf("source image has zero dimensions")
	}

	var scaledW, scaledH uint
	if srcW*height > srcH*width {
		scaledH = height
		scaledW = srcW * height / srcH
	} else {
		scaledW = width
		scaledH = srcH * width / srcW
	}

	scaled := resize.Resize(scaledW, scaledH, img, resize.Lanczos3)

	scaledBounds := scaled.Bounds()
	x0 := (scaledBounds.Dx() - int(width)) / 2
	y0 := (scaledBounds.Dy() - int(height)) / 2

	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}
	if si, ok := scaled.(subImager); ok {
		return si.SubImage(image.Rect(x0, y0, x0+int(width), y0+int(height))), "jpeg", nil
	}

	cropped := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))
	for y := 0; y < int(height); y++ {
		for x := 0; x < int(width); x++ {
			cropped.Set(x, y, scaled.At(x0+x, y0+y))
		}
	}
	return cropped, "jpeg", nil
}
