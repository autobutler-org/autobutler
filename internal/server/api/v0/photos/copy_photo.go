package v0_photos

import (
	"errors"
	"fmt"

	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/photoutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
	"github.com/autobutler-org/quark/pkg/vfs"
	"github.com/gin-gonic/gin"
)

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
				newRelPath, err := photoutil.CopyPhotoVFS(c.Request.Context(), fsys, req.RelPath)
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

var copyPhotoRoute = serverutil.ApiRoute("POST", "/photos/copy", copyPhoto)
