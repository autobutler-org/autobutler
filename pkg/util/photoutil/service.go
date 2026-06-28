package photoutil

import (
	"fmt"
	"image"
	"path/filepath"
	"strings"

	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
)

// GenerateThumbnailParams contains parameters for generating a thumbnail
type GenerateThumbnailParams struct {
	FilePath string
	Width    uint
	Height   uint
}

// GenerateThumbnailResult contains the result of generating a thumbnail
type GenerateThumbnailResult struct {
	Thumbnail image.Image
	Format    string
}

// GenerateThumbnail creates a thumbnail image from a source file.
// Supports both image and video files (video requires ffmpeg).
func GenerateThumbnail(params GenerateThumbnailParams) (*GenerateThumbnailResult, error) {
	ext := strings.ToLower(filepath.Ext(params.FilePath))
	fileType := storageutil.DetermineFileTypeFromPath("file" + ext)

	if fileType != storageutil.FileTypeImage && fileType != storageutil.FileTypeVideo {
		return nil, fmt.Errorf("unsupported file type for thumbnail: %s", ext)
	}

	if fileType == storageutil.FileTypeVideo {
		if !IsFFmpegAvailable() {
			return nil, fmt.Errorf("ffmpeg is required for video thumbnails but was not found on PATH")
		}
		thumbnail, err := VideoToThumbnail(params.FilePath, params.Width, params.Height)
		if err != nil {
			return nil, fmt.Errorf("failed to generate video thumbnail: %w", err)
		}
		return &GenerateThumbnailResult{
			Thumbnail: thumbnail,
			Format:    "jpeg",
		}, nil
	}

	// RAW camera files can't be decoded by Go's image package — convert
	// via an external tool first, then thumbnail the resulting JPEG.
	if IsRawFile(params.FilePath) {
		img, err := RawToJPEG(params.FilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to convert RAW file: %w", err)
		}
		cropped, _, cropErr := cropToFit(img, params.Width, params.Height)
		if cropErr != nil {
			return nil, fmt.Errorf("crop RAW thumbnail: %w", cropErr)
		}
		return &GenerateThumbnailResult{Thumbnail: cropped, Format: "jpeg"}, nil
	}

	thumbnail, format, err := ImageToThumbnail(params.FilePath, params.Width, params.Height)
	if err != nil {
		return nil, fmt.Errorf("failed to generate thumbnail: %w", err)
	}

	return &GenerateThumbnailResult{
		Thumbnail: thumbnail,
		Format:    format,
	}, nil
}
