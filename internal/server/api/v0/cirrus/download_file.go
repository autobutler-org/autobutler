package v0_files

import (
	"archive/zip"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/photoutil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
	"github.com/autobutler-org/autobutler/pkg/vfs"

	_ "github.com/gen2brain/heic"
	"github.com/gin-gonic/gin"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// downloadCirrusFile godoc
// @Summary Download a file or folder
// @Description Downloads a single file or zips a folder and streams it back to the client
// @Tags cirrus
// @Produce application/octet-stream
// @Param filePath query string false "File path to download"
// @Param serial query string false "Device serial number to filter by"
// @Param format query string false "Output format conversion (e.g. 'jpeg' to convert HEIC to JPEG)"
// @Success 200 {file} file
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /cirrus/download [get]
func downloadFile(c *gin.Context) *serverutil.Response {
	filePath := c.Query("filePath")
	serial := c.Query("serial")
	wantsJPEG := c.Query("format") == "jpeg"

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	// VFS path: only when serial is empty (no device routing needed) and not a
	// RAW file needing OS-path conversion. RAW → JPEG requires dcraw/LibRaw which
	// only works with a real filesystem path, so those always fall through to
	// StorageService even when VFS is present.
	if serial == "" && !(wantsJPEG && photoutil.IsRawFile(filePath)) {
		if reg := deps.VFSRegistry(); reg != nil {
			if fsys, ok := reg.Get("files"); ok {
				return downloadFileVFS(c, deps, fsys, filePath, wantsJPEG)
			}
		}
	}

	// StorageService fallback (serial routing, RAW conversion, etc.)
	result, err := deps.StorageService().DownloadFile(storageutil.DownloadFileParams{
		FilePath:     filePath,
		DeviceSerial: serial,
	})
	if err != nil {
		return serverutil.NotFound(err)
	}

	if result.IsFolder {
		zipWriter := zip.NewWriter(c.Writer)
		defer zipWriter.Close()
		dirFs := os.DirFS(result.FullPath)
		if err := zipWriter.AddFS(dirFs); err != nil {
			return serverutil.InternalServerError(fmt.Errorf("failed to zip folder: %w", err))
		}
		c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", filepath.Base(result.FullPath)))
		c.Writer.Header().Set("Content-Type", "application/octet-stream")
		return nil // response written directly to writer
	}

	f, err := os.Open(result.FullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return serverutil.NotFound(fmt.Errorf("file not found: %s", filePath))
		}
		return serverutil.InternalServerError(fmt.Errorf("failed to open file: %w", err))
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(result.FullPath))

	if wantsJPEG && result.FileType == storageutil.FileTypeImage {
		// Acquire IO semaphore: JPEG conversion is the most memory-intensive IO
		// path (full uncompressed image.Image decode + re-encode). Limit
		// concurrency to prevent RAM spikes and disk thrashing under concurrent
		// load — especially on spinning HDDs.
		if sem := deps.IOSemaphore(); sem != nil {
			if !sem.AcquireDefault(c.Request.Context()) {
				slog.Warn("download: IO semaphore timed out for JPEG conversion",
					"path", result.FullPath,
					"available", sem.Available(),
					"cap", sem.Cap(),
				)
				c.Header("Retry-After", "5")
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "server busy, please retry"})
				return nil
			}
			defer sem.Release()
		}

		baseName := strings.TrimSuffix(filepath.Base(result.FullPath), ext) + ".jpg"

		if photoutil.IsRawFile(result.FullPath) {
			jpegBytes, err := photoutil.RawToJPEGBytes(result.FullPath, 92)
			if err != nil {
				return serverutil.InternalServerError(fmt.Errorf("failed to convert RAW to JPEG: %w", err))
			}
			c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", baseName))
			c.Data(http.StatusOK, "image/jpeg", jpegBytes)
			return nil
		}

		img, _, err := image.Decode(f)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("failed to decode image: %w", err))
		}

		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", baseName))
		c.Header("Content-Type", "image/jpeg")
		c.Status(http.StatusOK)
		if err := jpeg.Encode(c.Writer, img, &jpeg.Options{Quality: 92}); err != nil {
			// Headers already committed; log only.
			slog.Error("download: JPEG stream encode failed", "path", result.FullPath, "err", err)
		}
		return nil
	}

	f.Close() // close before c.File re-opens it
	disposition := "inline"
	contentType := "application/octet-stream"
	if result.FileType == storageutil.FileTypePDF {
		contentType = "application/pdf"
	} else if result.FileType == storageutil.FileTypeImage {
		contentType = storageutil.ImageMIMETypeFromExtension(filepath.Ext(result.FullPath))
	} else if result.FileType == storageutil.FileTypeVideo {
		contentType = storageutil.VideoMIMETypeFromExtension(filepath.Ext(result.FullPath))
	} else if result.FileType == storageutil.FileTypeAudio {
		contentType = storageutil.AudioMIMETypeFromExtension(filepath.Ext(result.FullPath))
	}
	c.Header("Content-Disposition", fmt.Sprintf("%s; filename=%s", disposition, filepath.Base(result.FullPath)))
	c.Header("Content-Type", contentType)
	c.File(result.FullPath)
	return nil // response written directly via c.File
}

// downloadFileVFS handles file downloads via the VFS layer.
// RAW files (needing OS path for dcraw/LibRaw) are excluded before calling this.
func downloadFileVFS(c *gin.Context, deps deputil.Dependencies, fsys vfs.VFS, filePath string, wantsJPEG bool) *serverutil.Response {
	ctx := c.Request.Context()

	fi, err := fsys.Stat(ctx, filePath)
	if err != nil {
		return serverutil.NotFound(err)
	}

	if fi.IsDir {
		// Zip and stream the directory contents.
		c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", fi.Name))
		c.Writer.Header().Set("Content-Type", "application/octet-stream")
		zipWriter := zip.NewWriter(c.Writer)
		defer zipWriter.Close()

		entries, err := fsys.List(ctx, filePath, &vfs.ListFilter{Recursive: true})
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("failed to list folder: %w", err))
		}
		for _, entry := range entries {
			if entry.IsDir {
				continue
			}
			r, err := fsys.Open(ctx, entry.Path)
			if err != nil {
				return serverutil.InternalServerError(fmt.Errorf("failed to open %s: %w", entry.Path, err))
			}
			// Compute a relative path inside the zip (trim the base filePath prefix).
			rel := strings.TrimPrefix(entry.Path, filePath)
			rel = strings.TrimPrefix(rel, "/")
			w, err := zipWriter.Create(rel)
			if err != nil {
				r.Close()
				return serverutil.InternalServerError(fmt.Errorf("failed to create zip entry %s: %w", rel, err))
			}
			if _, err := io.Copy(w, r); err != nil {
				r.Close()
				return serverutil.InternalServerError(fmt.Errorf("failed to write zip entry %s: %w", rel, err))
			}
			r.Close()
		}
		return nil
	}

	mimeType := fi.MimeType
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	if wantsJPEG && strings.HasPrefix(mimeType, "image/") {
		// Acquire IO semaphore for JPEG conversion.
		if sem := deps.IOSemaphore(); sem != nil {
			if !sem.AcquireDefault(ctx) {
				slog.Warn("download: IO semaphore timed out for VFS JPEG conversion",
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

		r, err := fsys.Open(ctx, filePath)
		if err != nil {
			return serverutil.NotFound(err)
		}
		defer r.Close()

		img, _, err := image.Decode(r)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("failed to decode image: %w", err))
		}

		ext := strings.ToLower(filepath.Ext(filePath))
		baseName := strings.TrimSuffix(filepath.Base(filePath), ext) + ".jpg"
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", baseName))
		c.Header("Content-Type", "image/jpeg")
		c.Status(http.StatusOK)
		if err := jpeg.Encode(c.Writer, img, &jpeg.Options{Quality: 92}); err != nil {
			slog.Error("download: VFS JPEG stream encode failed", "path", filePath, "err", err)
		}
		return nil
	}

	// Use http.ServeContent when the VFS supports seeking — this enables HTTP
	// range requests (RFC 7233) for video/audio files, allowing seeking and
	// resumable downloads. Falls back to sequential DataFromReader otherwise.
	if seeker, ok := fsys.(vfs.Seeker); ok {
		rs, err := seeker.OpenSeeker(ctx, filePath)
		if err != nil {
			return serverutil.NotFound(err)
		}
		defer rs.Close()
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", fi.Name))
		c.Header("Content-Type", mimeType)
		http.ServeContent(c.Writer, c.Request, fi.Name, fi.ModTime, rs)
		return nil
	}

	r, err := fsys.Open(ctx, filePath)
	if err != nil {
		return serverutil.NotFound(err)
	}
	defer r.Close()

	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", fi.Name))
	c.DataFromReader(http.StatusOK, fi.Size, mimeType, r, nil)
	return nil
}

var downloadFileRoute = serverutil.ApiRoute(
	"GET", "/cirrus/download", downloadFile,
)
