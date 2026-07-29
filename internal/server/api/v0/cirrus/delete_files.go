package v0_files

import (
	"context"
	"errors"
	"log"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/eventbus"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
	"github.com/autobutler-org/autobutler/pkg/vfs"

	"github.com/gin-gonic/gin"
)

// deleteFiles godoc
// @Summary Delete files
// @Description Enqueue deletion of files under the specified root directory
// @Tags cirrus
// @Produce json
// @Param rootDir query string false "Root directory"
// @Param filePaths query []string true "Array of file paths to delete"
// @Param serial query string false "Device serial number to filter by"
// @Success 202 {object} serverutil.Response "Ok"
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /cirrus [delete]
func deleteFiles(c *gin.Context) *serverutil.Response {
	rootDir := c.Query("rootDir")
	filePaths := c.QueryArray("filePaths")
	serial := c.Query("serial")

	if len(filePaths) == 0 {
		return serverutil.BadRequest(errors.New("filePaths query parameter is required and must contain at least one file path"))
	}

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	// Use VFS.Delete when no serial and VFS is available (routes through StorageServiceVFS).
	usedVFS := false
	if serial == "" {
		if reg := deps.VFSRegistry(); reg != nil {
			if fsys, ok := reg.Get("files"); ok {
				for _, p := range filePaths {
					if err := fsys.Delete(c.Request.Context(), p, vfs.DeleteOptions{Recursive: true}); err != nil && err != vfs.ErrNotFound {
						return serverutil.InternalServerError(err)
					}
				}
				usedVFS = true
			}
		}
	}
	if !usedVFS {
		if _, err := deps.StorageService().DeleteFiles(storageutil.DeleteFilesParams{
			RootDir:      rootDir,
			FilePaths:    filePaths,
			DeviceSerial: serial,
		}); err != nil {
			return serverutil.InternalServerError(err)
		}
	}

	for _, p := range filePaths {
		deps.EventBus().Publish(eventbus.Event{
			Kind:         eventbus.EventDelete,
			Path:         p,
			DeviceSerial: serial,
		})
		// Clean up all DB records for the deleted photo regardless of whether a
		// serial is present (empty serial is a valid key in both tables).
		if database := deps.Database(); database != nil {
			if err := database.Queries.DeletePhotoFromAllAlbums(context.Background(), db.DeletePhotoFromAllAlbumsParams{
				DeviceSerial: serial,
				RelPath:      p,
			}); err != nil {
				log.Printf("autobutler: delete cleanup: remove album items for %q (serial=%q): %v", p, serial, err)
			}
			if err := database.Queries.DeletePhotoRotation(context.Background(), db.DeletePhotoRotationParams{
				DeviceSerial: serial,
				RelPath:      p,
			}); err != nil {
				log.Printf("autobutler: delete cleanup: remove rotation for %q (serial=%q): %v", p, serial, err)
			}
		}
	}
	return serverutil.Ok()
}

var deleteFilesRoute = serverutil.ApiRoute(
	"DELETE", "/cirrus", deleteFiles,
)
