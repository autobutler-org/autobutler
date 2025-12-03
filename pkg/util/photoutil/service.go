package photoutil

import (
	"fmt"
	"image"
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

// GenerateThumbnail creates a thumbnail image from a source file
func GenerateThumbnail(params GenerateThumbnailParams) (*GenerateThumbnailResult, error) {
	thumbnail, format, err := ImageToThumbnail(params.FilePath, params.Width, params.Height)
	if err != nil {
		return nil, fmt.Errorf("failed to generate thumbnail: %w", err)
	}

	return &GenerateThumbnailResult{
		Thumbnail: thumbnail,
		Format:    format,
	}, nil
}
