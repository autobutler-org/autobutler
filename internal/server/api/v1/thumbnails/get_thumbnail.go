package v1_thumbnails

import (
	"crypto/sha256"
	"fmt"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/photoutil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

const (
	thumbnailWidth  = 400
	thumbnailHeight = 400
)

// thumbnailCacheDir returns the path to the thumbnail cache directory,
// creating it if it doesn't exist.
func thumbnailCacheDir() (string, error) {
	cacheDir := filepath.Join(storageutil.GetDataDir(), "cache", "thumbnails")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create thumbnail cache directory: %w", err)
	}
	return cacheDir, nil
}

// cacheKey computes a SHA-256 hex digest of the given serial and file path,
// used as the filename in the cache directory.
func cacheKey(serial, filePath string) string {
	h := sha256.Sum256([]byte(serial + "/" + filePath))
	return fmt.Sprintf("%x", h)
}

// etagFromModTime returns a quoted ETag string derived from a file's modification time.
func etagFromModTime(t time.Time) string {
	return fmt.Sprintf(`"%x"`, t.UnixNano())
}

// contentTypeForExt returns the MIME type for a thumbnail based on its file extension.
func contentTypeForExt(ext string) string {
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return "image/jpeg"
	}
}

// getThumbnail godoc
// @Summary Get thumbnail for an image
// @Description Generates and returns a thumbnail (resized image) for the specified file
// @Tags thumbnails
// @Produce image/png image/jpeg
// @Param filePath path string true "Path to the image file"
// @Success 200 {file} file
// @Failure 304 "Not Modified"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /thumbnails/{filePath} [get]
var getThumbnailRoute = serverutil.ApiRoute(
	"GET", "/thumbnails/*filePath", func(c *gin.Context) *serverutil.Response {
		deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
		if !ok {
			return serverutil.InternalServerError(nil)
		}
		filePath := c.Param("filePath")
		// Default to the global cirrus directory but allow selecting a specific device by serial
		filesDir, err := storageutil.GetCirrusDir()
		if err != nil {
			return serverutil.InternalServerError(err)
		}
		serial := c.Query("serial")
		if serial != "" {
			if devices, err := deps.StorageService().GetManagedDevices(); err == nil {
				for _, d := range devices {
					if d.UsbInfo != nil && d.UsbInfo.GetSerial() == serial {
						filesDir = d.CirrusDir
						break
					}
				}
			}
		}

		fullPath := filepath.Join(filesDir, filePath)

		srcInfo, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			return serverutil.NotFound(fmt.Errorf("thumbnail not found: %s", filePath))
		}
		if err != nil {
			return serverutil.InternalServerError(err)
		}

		ext := strings.ToLower(filepath.Ext(filePath))

		// --- Disk cache lookup ---
		cacheDir, err := thumbnailCacheDir()
		if err != nil {
			return serverutil.InternalServerError(err)
		}
		key := cacheKey(serial, filePath)
		cachedPath := filepath.Join(cacheDir, key)

		cachedInfo, cacheErr := os.Stat(cachedPath)
		cacheHit := cacheErr == nil && cachedInfo.ModTime().After(srcInfo.ModTime())

		if !cacheHit {
			// Generate the thumbnail
			result, err := photoutil.GenerateThumbnail(photoutil.GenerateThumbnailParams{
				FilePath: fullPath,
				Width:    thumbnailWidth,
				Height:   thumbnailHeight,
			})
			if err != nil {
				return serverutil.InternalServerError(err)
			}

			// Write to cache file
			tmpPath := cachedPath + ".tmp"
			f, err := os.Create(tmpPath)
			if err != nil {
				return serverutil.InternalServerError(fmt.Errorf("failed to create cache file: %w", err))
			}

			switch ext {
			case ".png":
				err = png.Encode(f, result.Thumbnail)
			default:
				err = jpeg.Encode(f, result.Thumbnail, &jpeg.Options{Quality: 85})
			}
			f.Close()
			if err != nil {
				os.Remove(tmpPath)
				return serverutil.InternalServerError(fmt.Errorf("failed to encode thumbnail to cache: %w", err))
			}

			if err := os.Rename(tmpPath, cachedPath); err != nil {
				os.Remove(tmpPath)
				return serverutil.InternalServerError(fmt.Errorf("failed to commit cache file: %w", err))
			}

			// Re-stat the cache file for ETag
			cachedInfo, err = os.Stat(cachedPath)
			if err != nil {
				return serverutil.InternalServerError(err)
			}
		}

		// --- ETag / conditional response ---
		etag := etagFromModTime(cachedInfo.ModTime())
		c.Header("ETag", etag)
		c.Header("Cache-Control", "public, max-age=86400")
		c.Header("Content-Type", contentTypeForExt(ext))

		if match := c.GetHeader("If-None-Match"); match == etag {
			c.Status(http.StatusNotModified)
			return nil
		}

		// Serve the cached file
		data, err := os.ReadFile(cachedPath)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("failed to read cached thumbnail: %w", err))
		}
		c.Data(http.StatusOK, contentTypeForExt(ext), data)
		return nil
	},
)
