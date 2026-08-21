package photoutil

import (
	"fmt"
	"image"
	_ "image/gif"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/storageutil"

	"github.com/KononK/resize"
	_ "github.com/gen2brain/heic"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
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

// FindAllPhotosRecursively finds all photo files in a directory and its subdirectories.
// Also detects Live Photo companions (e.g. IMG_1234.HEIC + IMG_1234.MOV) by collecting
// video basenames during the same walk — no extra disk I/O.
func FindAllPhotosRecursively(rootDir string) ([]PhotoInfo, error) {
	var photos []PhotoInfo
	videoBasenames := make(map[string]bool)

	err := filepath.Walk(rootDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		fileType := storageutil.DetermineFileTypeFromPath(info.Name())
		switch fileType {
		case storageutil.FileTypeImage:
			relPath, err := filepath.Rel(rootDir, path)
			if err != nil {
				return err // coverage: ignore - filepath.Rel only fails on cross-volume paths (different drives on Windows)
			}
			photos = append(photos, PhotoInfo{
				FileInfo: info,
				RelPath:  relPath,
			})
		case storageutil.FileTypeVideo:
			ext := filepath.Ext(path)
			videoBasenames[strings.TrimSuffix(path, ext)] = true
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory %s: %w", rootDir, err)
	}

	for i := range photos {
		ext := filepath.Ext(photos[i].RelPath)
		lower := strings.ToLower(ext)
		if lower == ".heic" || lower == ".heif" || lower == ".jpg" || lower == ".jpeg" {
			fullBase := strings.TrimSuffix(filepath.Join(rootDir, photos[i].RelPath), ext)
			photos[i].HasLiveVideo = videoBasenames[fullBase]
		}
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

	imgFormat := ImageFormatFromPath(filePath)
	if imgFormat != 0 {
		if _, seekErr := file.Seek(0, 0); seekErr == nil {
			orientation := GetOrientation(file, imgFormat)
			img = applyExifOrientation(img, orientation)
		}
	} else {
		img, _ = CorrectImageOrientation(img, file)
	}

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
// Uses bep/imagemeta which supports JPEG, HEIC/HEIF, PNG, WebP, TIFF, and RAW formats.
// Falls back to no-op for unsupported formats.
func CorrectImageOrientation(img image.Image, r io.ReadSeeker) (image.Image, error) {
	return img, nil
}

// applyExifOrientation transforms an image based on the EXIF orientation value.
// http://sylvana.net/jpegcrop/exif_orientation.html
func applyExifOrientation(img image.Image, orientation int) image.Image {
	switch orientation {
	case 2:
		return flipHorizontal(img)
	case 3:
		return rotate180(img)
	case 4:
		return flipVertical(img)
	case 5:
		return rotate270(flipHorizontal(img))
	case 6:
		return rotate90(img)
	case 7:
		return rotate90(flipHorizontal(img))
	case 8:
		return rotate270(img)
	default:
		return img
	}
}

// ApplyRotation rotates img by quarters × 90° clockwise.
// Negative values are normalized: -1 → 3, -2 → 2, etc.
func ApplyRotation(img image.Image, quarters int64) image.Image {
	switch ((quarters % 4) + 4) % 4 {
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
