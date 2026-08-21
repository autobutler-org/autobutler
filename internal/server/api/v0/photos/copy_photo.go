package v0_photos

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
	"github.com/autobutler-org/quark/pkg/vfs"
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

	// VFS path: no-serial copies go through VFS.Open + VFS.Write.
	if req.Serial == "" {
		if reg := deps.VFSRegistry(); reg != nil {
			if fsys, ok := reg.Get("files"); ok {
				newRelPath, err := copyPhotoVFS(c.Request.Context(), fsys, req.RelPath)
				if err != nil {
					if errors.Is(err, vfs.ErrNotFound) {
						return serverutil.NotFound(fmt.Errorf("photo not found: %s", req.RelPath))
					}
					return serverutil.InternalServerError(err)
				}
				return serverutil.Ok().
					WithContentType(serverutil.ContentTypeJSON).
					WithData(copyPhotoResponse{RelPath: newRelPath})
			}
		}
	}

	// Fallback: StorageService.CopyFile for serial-scoped or non-VFS case.
	result, err := deps.StorageService().CopyFile(storageutil.CopyFileParams{
		RelPath:      req.RelPath,
		DeviceSerial: req.Serial,
	})
	if err != nil {
		if errors.Is(err, storageutil.ErrPathNotFound) {
			return serverutil.NotFound(fmt.Errorf("photo not found: %s", req.RelPath))
		}
		return serverutil.InternalServerError(err)
	}

	return serverutil.Ok().
		WithContentType(serverutil.ContentTypeJSON).
		WithData(copyPhotoResponse{RelPath: result.NewRelPath})
}

// copyPhotoVFS duplicates relPath within the VFS, returning the new relative path.
func copyPhotoVFS(ctx context.Context, fsys vfs.VFS, relPath string) (string, error) {
	// Verify source exists.
	if _, err := fsys.Stat(ctx, relPath); err != nil {
		return "", err
	}

	ext := filepath.Ext(relPath)
	stem := relPath[:len(relPath)-len(ext)]
	destPath := stem + "_copy" + ext

	// Find a non-conflicting destination name.
	for i := 2; i <= 100; i++ {
		if _, err := fsys.Stat(ctx, destPath); errors.Is(err, vfs.ErrNotFound) {
			break
		}
		destPath = fmt.Sprintf("%s_copy_%d%s", stem, i, ext)
	}

	// Copy: open source, write destination.
	rc, err := fsys.Open(ctx, relPath)
	if err != nil {
		return "", err
	}
	defer rc.Close()

	if err := fsys.Write(ctx, destPath, rc, vfs.WriteOptions{IfNoneMatch: "*"}); err != nil {
		return "", err
	}
	return destPath, nil
}

var copyPhotoRoute = serverutil.ApiRoute("POST", "/photos/copy", copyPhoto)
