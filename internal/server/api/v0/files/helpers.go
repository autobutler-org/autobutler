package v0_files

import (
	"archive/zip"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/uploadutil"
	"github.com/autobutler-org/quark/pkg/vfs"
	"github.com/gin-gonic/gin"
)

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

	r, err := fsys.Open(ctx, filePath)
	if err != nil {
		return serverutil.NotFound(err)
	}
	defer r.Close()

	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", fi.Name))
	c.Header("Content-Type", mimeType)

	// If the underlying VFS returns an io.ReadSeeker (e.g. *os.File from LocalVFS
	// or StorageServiceVFS), use http.ServeContent so the response honours HTTP
	// range requests (RFC 7233) — required for video seeking and resumable
	// downloads. Falls back to sequential streaming via DataFromReader otherwise.
	if rs, ok := r.(io.ReadSeeker); ok {
		http.ServeContent(c.Writer, c.Request, fi.Name, fi.ModTime, rs)
		return nil
	}

	c.DataFromReader(http.StatusOK, fi.Size, mimeType, r, nil)
	return nil
}

// uploadDestination is where an upload lands, for both this endpoint and the
// chunked sessions in upload_session.go. Both have to make the same choice
// between the VFS namespace and the StorageService, so the choice lives in one
// place (#1629).
func uploadDestination(deps deputil.Dependencies) uploadutil.Destination {
	return uploadutil.Destination{
		Registry: deps.VFSRegistry(),
		Storage:  deps.StorageService(),
		EventBus: deps.EventBus(),
	}
}

// Resumable chunked uploads (#1629). A large file no longer rides on one
// request that a dropped connection costs in full: the client opens a session,
// PUTs the bytes a chunk at a time, and after an interruption asks what landed
// and carries on from there. Small files still take the multipart endpoint in
// upload_files.go, where chunking would be pure overhead.
const (
	// uploadOffsetHeader carries the committed offset back with a 409, so a
	// client that guessed wrong resyncs without a second round trip.
	uploadOffsetHeader = "X-Upload-Offset"
	// sessionIDParam is the path parameter naming the session.
	sessionIDParam = "sessionId"
)

// uploadSessionError maps what uploadutil reports onto the status codes the
// client contract is written against. The offset mismatch is the only one that
// carries state back in a header: everything the client needs to resync from a
// 409 is in the response it already has.
func uploadSessionError(c *gin.Context, err error) *serverutil.Response {
	var mismatch *uploadutil.OffsetMismatchError
	switch {
	case errors.As(err, &mismatch):
		c.Header(uploadOffsetHeader, strconv.FormatInt(mismatch.Offset, 10))
		return serverutil.Conflict(err)
	case errors.Is(err, uploadutil.ErrSessionNotFound):
		return serverutil.NotFound(err)
	case errors.Is(err, uploadutil.ErrInvalidRange),
		errors.Is(err, uploadutil.ErrInvalidRequest),
		// The file already exists and the caller did not ask to overwrite it.
		// The multipart endpoint answers 400 for that, and the same upload
		// arriving in chunks should not answer something else.
		errors.Is(err, vfs.ErrConflict):
		return serverutil.BadRequest(err)
	default:
		return serverutil.InternalServerError(err)
	}
}
