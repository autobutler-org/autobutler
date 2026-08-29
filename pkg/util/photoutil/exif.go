package photoutil

import (
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strings"

	"github.com/bep/imagemeta"
)

// ImageFormatFromPath returns the imagemeta.ImageFormat for a file path,
// or 0 if the format is not supported.
func ImageFormatFromPath(filePath string) imagemeta.ImageFormat {
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".jpg", ".jpeg":
		return imagemeta.JPEG
	case ".heic", ".heif":
		return imagemeta.HEIF
	case ".png":
		return imagemeta.PNG
	case ".webp":
		return imagemeta.WebP
	case ".tiff", ".tif":
		return imagemeta.TIFF
	case ".avif":
		return imagemeta.AVIF
	case ".dng":
		return imagemeta.DNG
	case ".cr2":
		return imagemeta.CR2
	case ".nef":
		return imagemeta.NEF
	case ".arw":
		return imagemeta.ARW
	case ".orf":
		return imagemeta.TIFF
	case ".rw2":
		return imagemeta.TIFF
	default:
		return 0
	}
}

// DecodeExif extracts EXIF metadata from a ReadSeeker.
// The format must be provided (use ImageFormatFromPath).
// Returns nil if the format is unsupported or no EXIF data is found.
func DecodeExif(r io.ReadSeeker, format imagemeta.ImageFormat) (*ExifData, error) {
	if format == 0 {
		return nil, nil
	}

	var tags imagemeta.Tags
	result, err := imagemeta.Decode(imagemeta.Options{
		R:           r,
		ImageFormat: format,
		Sources:     imagemeta.EXIF | imagemeta.XMP | imagemeta.CONFIG,
		HandleTag: func(info imagemeta.TagInfo) error {
			tags.Add(info)
			return nil
		},
	})
	if err != nil {
		if imagemeta.IsInvalidFormat(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("decode metadata: %w", err)
	}

	exif := tags.EXIF()
	data := &ExifData{}
	empty := true

	if v, ok := exifUint16(exif, "Orientation"); ok {
		data.Orientation = int(v)
		empty = false
	}

	if t, err := tags.GetDateTime(); err == nil && !t.IsZero() {
		utc := t.UTC()
		data.DateTaken = &utc
		empty = false
	}

	if v, ok := exifString(exif, "Make"); ok {
		data.Make = v
		empty = false
	}
	if v, ok := exifString(exif, "Model"); ok {
		data.Model = v
		empty = false
	}
	if v, ok := exifString(exif, "LensModel"); ok {
		data.LensModel = v
		empty = false
	}

	if v, ok := exifFloat(exif, "FNumber"); ok {
		data.Aperture = v
		empty = false
	}

	if v, ok := exifRat(exif, "ExposureTime"); ok {
		data.ShutterSpeed = v
		empty = false
	}

	if v, ok := exifInt(exif, "ISO"); ok {
		data.ISO = int(v)
		empty = false
	}

	if v, ok := exifFloat(exif, "FocalLength"); ok {
		data.FocalLength = v
		empty = false
	}

	if lat, lon, err := tags.GetLatLong(); err == nil && (lat != 0 || lon != 0) {
		data.Latitude = lat
		data.Longitude = lon
		data.HasGPS = true
		empty = false
	}

	data.Width = result.ImageConfig.Width
	data.Height = result.ImageConfig.Height
	// For HEIC/HEIF and some other formats, imagemeta may not populate
	// ImageConfig dimensions. Fall back to EXIF PixelXDimension /
	// PixelYDimension tags which cameras always embed.
	if data.Width == 0 && data.Height == 0 {
		if v, ok := exifInt(exif, "PixelXDimension"); ok && v > 0 {
			data.Width = int(v)
		}
		if v, ok := exifInt(exif, "PixelYDimension"); ok && v > 0 {
			data.Height = int(v)
		}
	}
	if data.Width > 0 || data.Height > 0 {
		empty = false
	}

	if empty {
		return nil, nil
	}
	return data, nil
}

// GetOrientation extracts just the EXIF orientation from a ReadSeeker.
// Returns 1 (normal) if orientation cannot be determined.
func GetOrientation(r io.ReadSeeker, format imagemeta.ImageFormat) int {
	if format == 0 {
		return 1
	}

	orientation := 1
	imagemeta.Decode(imagemeta.Options{
		R:           r,
		ImageFormat: format,
		Sources:     imagemeta.EXIF,
		HandleTag: func(info imagemeta.TagInfo) error {
			if info.Tag == "Orientation" {
				if v, ok := info.Value.(uint16); ok {
					orientation = int(v)
					return imagemeta.ErrStopWalking
				}
			}
			return nil
		},
	})

	return orientation
}

func exifString(exif map[string]imagemeta.TagInfo, tag string) (string, bool) {
	ti, ok := exif[tag]
	if !ok {
		return "", false
	}
	s, ok := ti.Value.(string)
	if !ok || s == "" {
		return "", false
	}
	return strings.TrimSpace(s), true
}

func exifFloat(exif map[string]imagemeta.TagInfo, tag string) (float64, bool) {
	ti, ok := exif[tag]
	if !ok {
		return 0, false
	}
	switch v := ti.Value.(type) {
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, false
		}
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	default:
		return 0, false
	}
}

func exifInt(exif map[string]imagemeta.TagInfo, tag string) (int64, bool) {
	ti, ok := exif[tag]
	if !ok {
		return 0, false
	}
	switch v := ti.Value.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}

func exifUint16(exif map[string]imagemeta.TagInfo, tag string) (uint16, bool) {
	ti, ok := exif[tag]
	if !ok {
		return 0, false
	}
	switch v := ti.Value.(type) {
	case uint16:
		return v, true
	case int:
		return uint16(v), true
	case uint32:
		return uint16(v), true
	default:
		return 0, false
	}
}

func exifRat(exif map[string]imagemeta.TagInfo, tag string) ([2]int64, bool) {
	ti, ok := exif[tag]
	if !ok {
		return [2]int64{}, false
	}
	switch v := ti.Value.(type) {
	case float64:
		if v == 0 {
			return [2]int64{0, 1}, true
		}
		if v < 1 {
			denom := int64(math.Round(1.0 / v))
			return [2]int64{1, denom}, true
		}
		return [2]int64{int64(math.Round(v * 1000)), 1000}, true
	default:
		return [2]int64{}, false
	}
}
