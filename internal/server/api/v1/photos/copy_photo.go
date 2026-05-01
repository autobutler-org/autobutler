package v1_photos

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
	"github.com/gin-gonic/gin"
)

type copyPhotoRequest struct {
	RelPath string `json:"relPath" binding:"required"`
	Serial  string `json:"serial"`
}

type copyPhotoResponse struct {
	RelPath string `json:"relPath"`
}

// copyPhoto godoc
// @Summary Duplicate a photo file
// @Description Creates a copy of the photo in the same directory with a non-conflicting name (e.g. photo_copy.jpg).
// @Tags photos
// @Accept json
// @Produce json
// @Param body body copyPhotoRequest true "Copy request"
// @Success 200 {object} copyPhotoResponse
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /photos/copy [post]
func copyPhoto(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	var req copyPhotoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(fmt.Errorf("invalid request: %w", err))
	}

	// Resolve base directory (same pattern as get_metadata.go).
	filesDir, err := storageutil.GetCirrusDir()
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	if req.Serial != "" {
		if devices, err := deps.StorageService().GetManagedDevices(); err == nil {
			for _, d := range devices {
				if d.UsbInfo != nil && d.UsbInfo.GetSerial() == req.Serial {
					filesDir = d.CirrusDir
					break
				}
			}
		}
	}

	// Guard against path traversal.
	cleanFilesDir := filepath.Clean(filesDir)
	srcFull := filepath.Join(cleanFilesDir, req.RelPath)
	if !strings.HasPrefix(srcFull, cleanFilesDir+string(filepath.Separator)) {
		return serverutil.BadRequest(fmt.Errorf("invalid relPath"))
	}

	if _, err := os.Stat(srcFull); os.IsNotExist(err) {
		return serverutil.NotFound(fmt.Errorf("photo not found: %s", req.RelPath))
	} else if err != nil {
		return serverutil.InternalServerError(err)
	}

	// Build the destination path: <name>_copy.<ext>, resolving collisions via
	// GetNonConflictingPath (which appends _(n) for further conflicts).
	ext := filepath.Ext(srcFull)
	stem := srcFull[:len(srcFull)-len(ext)]
	destFull := storageutil.GetNonConflictingPath(stem + "_copy" + ext)

	if err := copyFile(srcFull, destFull); err != nil {
		return serverutil.InternalServerError(fmt.Errorf("copy failed: %w", err))
	}

	// Return the relative path of the new file.
	newRelPath, err := filepath.Rel(cleanFilesDir, destFull)
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	return serverutil.Ok().
		WithContentType(serverutil.ContentTypeJSON).
		WithData(copyPhotoResponse{RelPath: newRelPath})
}

// copyFile copies src to dst atomically enough for our purposes: write to dst
// directly (same filesystem, so partial writes are the only risk, which is
// acceptable for a user-initiated duplicate action).
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

var copyPhotoRoute = serverutil.ApiRoute("POST", "/photos/copy", copyPhoto)
