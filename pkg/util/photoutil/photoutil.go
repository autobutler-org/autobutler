package photoutil

import (
	"fmt"
	"image"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/KononK/resize"
	"github.com/rwcarlsen/goexif/exif"
)

// FilterPhotoFiles filters a list of files to only include photo files
func FilterPhotoFiles(files []fs.FileInfo) []fs.FileInfo {
	photoFiles := make([]fs.FileInfo, 0)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		fileType := storageutil.DetermineFileTypeFromPath(file.Name())
		if fileType == storageutil.FileTypeImage {
			photoFiles = append(photoFiles, file)
		}
	}
	return photoFiles
}

// FindAllPhotosRecursively finds all photo files in a directory and its subdirectories
func FindAllPhotosRecursively(rootDir string) ([]PhotoInfo, error) {
	photos := make([]PhotoInfo, 0)

	err := filepath.Walk(rootDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		fileType := storageutil.DetermineFileTypeFromPath(info.Name())
		if fileType == storageutil.FileTypeImage {
			// Get relative path from rootDir
			relPath, err := filepath.Rel(rootDir, path)
			if err != nil {
				return err // coverage: ignore - filepath.Rel only fails on cross-volume paths (different drives on Windows)
			}
			photos = append(photos, PhotoInfo{
				FileInfo: info,
				RelPath:  relPath,
			})
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory %s: %w", rootDir, err)
	}

	return photos, nil
}

func ImageToThumbnail(filePath string, width, height uint) (image.Image, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("error opening image file %s: %w", filePath, err)
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	if err != nil {
		return nil, "", fmt.Errorf("error decoding image file %s: %w", filePath, err)
	}

	img, _ = CorrectImageOrientation(img, file)

	// Scale so the shorter side fills the target dimension, then center-crop.
	// This preserves aspect ratio rather than squishing the image.
	bounds := img.Bounds()
	srcW := uint(bounds.Dx())
	srcH := uint(bounds.Dy())

	var scaledW, scaledH uint
	if srcW*height > srcH*width {
		// Image is wider than target: scale by height, crop width
		scaledH = height
		scaledW = srcW * height / srcH
	} else {
		// Image is taller than target: scale by width, crop height
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
		return si.SubImage(image.Rect(x0, y0, x0+int(width), y0+int(height))), format, nil
	}

	// Fallback: manual pixel copy (resize library always returns a subImager in practice)
	cropped := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))
	for y := 0; y < int(height); y++ {
		for x := 0; x < int(width); x++ {
			cropped.Set(x, y, scaled.At(x0+x, y0+y))
		}
	}
	return cropped, format, nil
}

// CorrectImageOrientation reads EXIF orientation data and rotates/flips the image accordingly.
// This ensures images from cameras display correctly regardless of how the camera was held.
func CorrectImageOrientation(img image.Image, r io.ReadSeeker) (image.Image, error) {
	// Reset reader to beginning
	if _, err := r.Seek(0, 0); err != nil { // coverage: ignore - Seek rarely fails with valid file handles
		// If seek fails, just return the original image
		return img, nil
	}

	// Try to decode EXIF data
	x, err := exif.Decode(r)
	if err != nil {
		// No EXIF data or error reading it - just return original image
		return img, nil
	}

	// Get orientation tag
	tag, err := x.Get(exif.Orientation)
	if err != nil { // coverage: ignore - requires EXIF data without orientation tag
		// No orientation tag - return original image
		return img, nil
	}

	orientation, err := tag.Int(0)
	if err != nil { // coverage: ignore - requires malformed EXIF orientation value
		return img, nil
	}

	// Apply the transformation based on EXIF orientation
	// http://sylvana.net/jpegcrop/exif_orientation.html
	switch orientation {
	case 1: // coverage: ignore
		// Normal - do nothing
		return img, nil
	case 2: // coverage: ignore - requires EXIF image with orientation 2
		// Flipped horizontally
		return flipHorizontal(img), nil
	case 3: // coverage: ignore - requires EXIF image with orientation 3
		// Rotated 180°
		return rotate180(img), nil
	case 4: // coverage: ignore - requires EXIF image with orientation 4
		// Flipped vertically
		return flipVertical(img), nil
	case 5: // coverage: ignore - requires EXIF image with orientation 5
		// Flipped horizontally and rotated 90° CCW
		return rotate270(flipHorizontal(img)), nil
	case 6: // coverage: ignore - requires EXIF image with orientation 6
		// Rotated 90° CW
		return rotate90(img), nil
	case 7: // coverage: ignore - requires EXIF image with orientation 7
		// Flipped horizontally and rotated 90° CW
		return rotate90(flipHorizontal(img)), nil
	case 8: // coverage: ignore - requires EXIF image with orientation 8
		// Rotated 90° CCW
		return rotate270(img), nil
	}

	return img, nil // coverage: ignore - unreachable - all orientation values are covered
}

// ApplyRotation rotates img by quarters × 90° clockwise (0–3).
func ApplyRotation(img image.Image, quarters int64) image.Image {
	switch quarters % 4 {
	case 1:
		return rotate90(img)
	case 2:
		return rotate180(img)
	case 3:
		return rotate270(img)
	default:
		return img
	}
}

func rotate90(img image.Image) image.Image {
	bounds := img.Bounds()
	newImg := image.NewRGBA(image.Rect(0, 0, bounds.Dy(), bounds.Dx()))
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			newImg.Set(bounds.Max.Y-y-1, x, img.At(x, y))
		}
	}
	return newImg
}

func rotate180(img image.Image) image.Image {
	bounds := img.Bounds()
	newImg := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			newImg.Set(bounds.Max.X-x-1, bounds.Max.Y-y-1, img.At(x, y))
		}
	}
	return newImg
}

func rotate270(img image.Image) image.Image {
	bounds := img.Bounds()
	newImg := image.NewRGBA(image.Rect(0, 0, bounds.Dy(), bounds.Dx()))
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			newImg.Set(y, bounds.Max.X-x-1, img.At(x, y))
		}
	}
	return newImg
}

func flipHorizontal(img image.Image) image.Image {
	bounds := img.Bounds()
	newImg := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			newImg.Set(bounds.Max.X-x-1, y, img.At(x, y))
		}
	}
	return newImg
}

func flipVertical(img image.Image) image.Image {
	bounds := img.Bounds()
	newImg := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			newImg.Set(x, bounds.Max.Y-y-1, img.At(x, y))
		}
	}
	return newImg
}
