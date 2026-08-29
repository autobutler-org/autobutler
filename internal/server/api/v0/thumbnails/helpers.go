package v0_thumbnails

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"image/jpeg"
	"image/png"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/photoutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/vfs"
	"github.com/gin-gonic/gin"
)

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
