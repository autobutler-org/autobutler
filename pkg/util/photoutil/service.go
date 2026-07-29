package photoutil

import (
	"bytes"
	"fmt"
	"image"
	"io"
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

// GenerateThumbnailFromReader creates a thumbnail from an io.Reader.
// ext is the lowercase file extension (e.g. ".jpg") used for format detection.
// RAW and video files are not supported — callers must use GenerateThumbnail for those.
func GenerateThumbnailFromReader(r io.Reader, ext string, width, height uint) (*GenerateThumbnailResult, error) {
	fileType := storageutil.DetermineFileTypeFromPath("file" + ext)
	if fileType != storageutil.FileTypeImage {
		return nil, fmt.Errorf("GenerateThumbnailFromReader: unsupported file type for extension %q", ext)
	}

	// Buffer the reader so we can seek back for EXIF after image decode.
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("GenerateThumbnailFromReader: read: %w", err)
	}
	rs := bytes.NewReader(data)

	img, format, err := image.Decode(rs)
	if err != nil {
		return nil, fmt.Errorf("GenerateThumbnailFromReader: decode: %w", err)
	}

	// Seek back and apply EXIF orientation.
	if _, seekErr := rs.Seek(0, 0); seekErr == nil {
		imgFormat := ImageFormatFromPath("file" + ext)
		if imgFormat != 0 {
			orientation := GetOrientation(rs, imgFormat)
			img = applyExifOrientation(img, orientation)
		}
	}

	cropped, _, err := cropToFit(img, width, height)
	if err != nil {
		return nil, fmt.Errorf("GenerateThumbnailFromReader: crop: %w", err)
	}
	return &GenerateThumbnailResult{Thumbnail: cropped, Format: format}, nil
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
