package v0_thumbnails

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"image/jpeg"
	"image/png"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/photoutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
	"github.com/autobutler-org/quark/pkg/util/videoutil"
	"github.com/autobutler-org/quark/pkg/vfs"

	"github.com/gin-gonic/gin"
)

// thumbnailSize represents the supported thumbnail size tiers.
type thumbnailSize string

const (
	thumbnailSizeSm thumbnailSize = "sm" // 96×96  – grid icons / file browser
	thumbnailSizeMd thumbnailSize = "md" // 240×240 – card previews
	thumbnailSizeLg thumbnailSize = "lg" // 400×400 – detail view (legacy default)
)

// thumbnailDimensions returns the pixel dimensions for a given size tier.
func thumbnailDimensions(size thumbnailSize) (width, height uint) {
	switch size {
	case thumbnailSizeSm:
		return 96, 96
	case thumbnailSizeMd:
		return 240, 240
	default: // thumbnailSizeLg and any unknown value
		return 400, 400
	}
}

// parseThumbnailSize parses the ?size= query parameter, defaulting to lg.
func parseThumbnailSize(raw string) thumbnailSize {
	switch thumbnailSize(raw) {
	case thumbnailSizeSm:
		return thumbnailSizeSm
	case thumbnailSizeMd:
		return thumbnailSizeMd
	default:
		return thumbnailSizeLg
	}
}

// thumbnailCacheDir returns the path to the thumbnail cache directory,
// creating it if it doesn't exist.
func thumbnailCacheDir() (string, error) {
	cacheDir := filepath.Join(storageutil.GetDataDir(), "cache", "thumbnails")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create thumbnail cache directory: %w", err)
	}
	return cacheDir, nil
}

// thumbnailCacheVersion is bumped whenever the thumbnail generation algorithm
// changes, so that stale cached thumbnails are automatically regenerated.
const thumbnailCacheVersion = "v2"

// cacheKey computes a SHA-256 hex digest of the given serial, file path,
// rotation, and size so that each combination produces a distinct cache entry.
func cacheKey(serial, filePath string, rotationQuarters int64, size thumbnailSize) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%s/%s:r%d:s%s", thumbnailCacheVersion, serial, filePath, rotationQuarters, size)))
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
// @Param serial query string false "Device serial number (for device-specific files)"
// @Param size query string false "Thumbnail size tier: sm (96px), md (240px), lg (400px). Defaults to lg." Enums(sm, md, lg)
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
		serial := c.Query("serial")
		ext := strings.ToLower(filepath.Ext(filePath))
		fileType := storageutil.DetermineFileTypeFromPath("file" + ext)
		if fileType != storageutil.FileTypeImage && fileType != storageutil.FileTypeVideo {
			return serverutil.NotFound(fmt.Errorf("no thumbnail for file type %q: %s", fileType, filePath))
		}
		isVideo := fileType == storageutil.FileTypeVideo

		// relPath strips the leading '/' that the wildcard param includes.
		relPath := strings.TrimPrefix(filePath, "/")

		// VFS path: no-serial, non-RAW, non-video images only.
		// RAW and video need OS paths for external tools (dcraw/ffmpeg).
		if serial == "" && !isVideo && !photoutil.IsRawFile(relPath) {
			if reg := deps.VFSRegistry(); reg != nil {
				if fsys, ok := reg.Get("files"); ok {
					if resp := getThumbnailVFS(c, deps, fsys, relPath, ext, filePath, serial); resp != vfsThumbnailFallthrough {
						return resp
					}
				}
			}
		}

		// StorageService fallback: serial-scoped, RAW, video, or no VFS.
		filesDir, err := storageutil.GetCirrusDir()
		if err != nil {
			return serverutil.InternalServerError(err)
		}
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

		// --- Server-side rotation ---
		var rotationQuarters int64
		if rq, err := deps.Database().Queries.GetPhotoRotation(
			context.Background(),
			db.GetPhotoRotationParams{DeviceSerial: serial, RelPath: relPath},
		); err == nil {
			rotationQuarters = rq
		} else if !errors.Is(err, sql.ErrNoRows) {
			return serverutil.InternalServerError(fmt.Errorf("get photo rotation: %w", err))
		}

		// --- Size parameter ---
		size := parseThumbnailSize(c.Query("size"))
		width, height := thumbnailDimensions(size)

		// --- Disk cache lookup ---
		// Rotation and size are included in the key so each combination gets
		// its own cache entry.
		cacheDir, err := thumbnailCacheDir()
		if err != nil {
			return serverutil.InternalServerError(err)
		}
		key := cacheKey(serial, filePath, rotationQuarters, size)
		cachedPath := filepath.Join(cacheDir, key)

		cachedInfo, cacheErr := os.Stat(cachedPath)
		cacheHit := cacheErr == nil && cachedInfo.ModTime().After(srcInfo.ModTime())

		if !cacheHit {
			// Acquire IO semaphore before disk-bound thumbnail generation.
			if sem := deps.IOSemaphore(); sem != nil {
				if !sem.AcquireDefault(c.Request.Context()) {
					slog.Warn("thumbnail: IO semaphore timed out",
						"path", filePath,
						"available", sem.Available(),
						"cap", sem.Cap(),
					)
					c.Header("Retry-After", "5")
					c.JSON(http.StatusServiceUnavailable, gin.H{"error": "server busy, please retry"})
					return nil
				}
				defer sem.Release()
			}

			// For video files: extract a representative frame via ffmpeg, then
			// feed that JPEG through the normal photoutil thumbnail pipeline.
			thumbSrcPath := fullPath
			var videoFrameTmp string
			if isVideo {
				if !videoutil.Available() {
					return serverutil.NotFound(fmt.Errorf("video thumbnails require ffmpeg (not installed)"))
				}
				// Probe to pick a good timestamp (2s or 10% of duration).
				probeCtx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
				defer cancel()
				seekTs := 2 * time.Second
				if info, probeErr := videoutil.Probe(probeCtx, fullPath); probeErr == nil {
					if tenth := info.Duration / 10; info.Duration < 20*time.Second && tenth < seekTs {
						seekTs = tenth
					}
				}
				tmpFile, tmpErr := os.CreateTemp("", "vthumb-*.jpg")
				if tmpErr != nil {
					return serverutil.InternalServerError(fmt.Errorf("video thumb temp file: %w", tmpErr))
				}
				tmpFile.Close()
				videoFrameTmp = tmpFile.Name()
				defer os.Remove(videoFrameTmp)
				extractCtx, extractCancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
				defer extractCancel()
				if extractErr := videoutil.ExtractFrame(extractCtx, fullPath, seekTs, videoFrameTmp); extractErr != nil {
					return serverutil.InternalServerError(fmt.Errorf("extract video frame: %w", extractErr))
				}
				thumbSrcPath = videoFrameTmp
			}

			// Generate the thumbnail at the requested size.
			result, err := photoutil.GenerateThumbnail(photoutil.GenerateThumbnailParams{
				FilePath: thumbSrcPath,
				Width:    width,
				Height:   height,
			})
			if err != nil {
				return serverutil.InternalServerError(err)
			}

			// Apply server-side rotation so the cached thumbnail matches the
			// orientation the user has set.
			if rotationQuarters != 0 {
				result.Thumbnail = photoutil.ApplyRotation(result.Thumbnail, rotationQuarters)
			}

			// Compute and store perceptual dHash async — the image is already
			// decoded here so hashing is nearly free. Non-blocking; a failure to
			// store the hash does not affect thumbnail delivery.
			if !isVideo {
				thumb := result.Thumbnail
				go func() {
					hashHex := photoutil.DHashHex(thumb)
					_ = deps.Database().Queries.UpsertPhotoHash(
						context.Background(),
						db.UpsertPhotoHashParams{
							DeviceSerial: serial,
							RelPath:      relPath,
							Dhash:        sql.NullString{String: hashHex, Valid: true},
						},
					)
				}()
			}

			// Write to cache file
			tmpPath := cachedPath + ".tmp"
			f, err := os.Create(tmpPath)
			if err != nil {
				return serverutil.InternalServerError(fmt.Errorf("failed to create cache file: %w", err))
			}

			if !isVideo && ext == ".png" {
				err = png.Encode(f, result.Thumbnail)
			} else {
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
		// no-cache: the browser must revalidate every request via If-None-Match.
		// The ETag covers rotation state (cache key includes rotationQuarters),
		// so the browser gets fresh bytes immediately after a rotation without
		// waiting for a max-age window to expire.
		c.Header("Cache-Control", "no-cache")
		thumbContentType := contentTypeForExt(ext)
		if isVideo {
			thumbContentType = "image/jpeg"
		}
		c.Header("Content-Type", thumbContentType)

		if match := c.GetHeader("If-None-Match"); match == etag {
			c.Status(http.StatusNotModified)
			return nil
		}

		// Serve the cached file
		data, err := os.ReadFile(cachedPath)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("failed to read cached thumbnail: %w", err))
		}
		c.Data(http.StatusOK, thumbContentType, data)
		return nil
	},
)

// vfsThumbnailFallthrough is a sentinel returned by getThumbnailVFS to signal
// that the VFS path could not serve the thumbnail and the caller should fall
// through to the StorageService path.
var vfsThumbnailFallthrough = &serverutil.Response{}

// getThumbnailVFS attempts to serve a thumbnail via the VFS. Returns
// vfsThumbnailFallthrough if the VFS is unable to handle the request (file not
// found, unsupported type, etc.) so the caller can use the StorageService path.
func getThumbnailVFS(
	c *gin.Context,
	deps deputil.Dependencies,
	fsys vfs.VFS,
	relPath, ext, filePath, serial string,
) *serverutil.Response {
	ctx := c.Request.Context()

	fi, err := fsys.Stat(ctx, relPath)
	if err != nil {
		// File not accessible via VFS — fall through.
		return vfsThumbnailFallthrough
	}

	// --- Server-side rotation ---
	var rotationQuarters int64
	if rq, err := deps.Database().Queries.GetPhotoRotation(
		context.Background(),
		db.GetPhotoRotationParams{DeviceSerial: serial, RelPath: relPath},
	); err == nil {
		rotationQuarters = rq
	} else if !errors.Is(err, sql.ErrNoRows) {
		return serverutil.InternalServerError(fmt.Errorf("get photo rotation: %w", err))
	}

	// --- Size parameter ---
	size := parseThumbnailSize(c.Query("size"))
	width, height := thumbnailDimensions(size)

	// --- Disk cache lookup ---
	cacheDir, err := thumbnailCacheDir()
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	key := cacheKey(serial, filePath, rotationQuarters, size)
	cachedPath := filepath.Join(cacheDir, key)

	cachedInfo, cacheErr := os.Stat(cachedPath)
	cacheHit := cacheErr == nil && cachedInfo.ModTime().After(fi.ModTime)

	if !cacheHit {
		if sem := deps.IOSemaphore(); sem != nil {
			if !sem.AcquireDefault(c.Request.Context()) {
				slog.Warn("thumbnail: IO semaphore timed out (VFS path)",
					"path", filePath,
					"available", sem.Available(),
					"cap", sem.Cap(),
				)
				c.Header("Retry-After", "5")
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "server busy, please retry"})
				return nil
			}
			defer sem.Release()
		}

		r, openErr := fsys.Open(ctx, relPath)
		if openErr != nil {
			return vfsThumbnailFallthrough
		}
		defer r.Close()

		result, genErr := photoutil.GenerateThumbnailFromReader(r, ext, width, height)
		if genErr != nil {
			// Unsupported format or decode error — fall through to StorageService.
			return vfsThumbnailFallthrough
		}

		if rotationQuarters != 0 {
			result.Thumbnail = photoutil.ApplyRotation(result.Thumbnail, rotationQuarters)
		}

		tmpPath := cachedPath + ".tmp"
		f, createErr := os.Create(tmpPath)
		if createErr != nil {
			return serverutil.InternalServerError(fmt.Errorf("failed to create cache file: %w", createErr))
		}
		var encErr error
		if ext == ".png" {
			encErr = png.Encode(f, result.Thumbnail)
		} else {
			encErr = jpeg.Encode(f, result.Thumbnail, &jpeg.Options{Quality: 85})
		}
		f.Close()
		if encErr != nil {
			os.Remove(tmpPath)
			return serverutil.InternalServerError(fmt.Errorf("failed to encode thumbnail: %w", encErr))
		}
		if renErr := os.Rename(tmpPath, cachedPath); renErr != nil {
			os.Remove(tmpPath)
			return serverutil.InternalServerError(fmt.Errorf("failed to commit cache file: %w", renErr))
		}
		cachedInfo, err = os.Stat(cachedPath)
		if err != nil {
			return serverutil.InternalServerError(err)
		}
	}

	// --- ETag / conditional response ---
	etag := etagFromModTime(cachedInfo.ModTime())
	c.Header("ETag", etag)
	c.Header("Cache-Control", "no-cache")
	thumbContentType := contentTypeForExt(ext)
	c.Header("Content-Type", thumbContentType)

	if match := c.GetHeader("If-None-Match"); match == etag {
		c.Status(http.StatusNotModified)
		return nil
	}

	data, readErr := os.ReadFile(cachedPath)
	if readErr != nil {
		return serverutil.InternalServerError(fmt.Errorf("failed to read cached thumbnail: %w", readErr))
	}
	c.Data(http.StatusOK, thumbContentType, data)
	return nil
}
