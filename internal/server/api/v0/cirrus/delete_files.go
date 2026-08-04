package v0_files

import (
	"context"
	"errors"
	"log"
	"time"

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
// @Description Soft-deletes files by moving them to the device trash directory (.trash/).
// @Description The operation returns 202 immediately after the rename — files are invisible
// @Description to listings at once. DB cleanup (album items, rotations, favorites) runs in
// @Description the background and completes within seconds. VFS-backed paths fall back to
// @Description hard delete via the plugin.
// @Tags cirrus
// @Produce json
// @Param rootDir query string false "Root directory"
// @Param filePaths query []string true "Array of file paths to delete"
// @Param serial query string false "Device serial number to filter by"
// @Success 202 {object} serverutil.Response "Accepted"
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

	// VFS path: plugin owns the delete semantics — no soft-delete wrapping.
	if serial == "" {
		if reg := deps.VFSRegistry(); reg != nil {
			if fsys, ok := reg.Get("files"); ok {
				for _, p := range filePaths {
					if err := fsys.Delete(c.Request.Context(), p, vfs.DeleteOptions{Recursive: true}); err != nil && err != vfs.ErrNotFound {
						return serverutil.InternalServerError(err)
					}
				}
				publishAndCleanup(deps, filePaths, serial)
				return serverutil.Accepted()
			}
		}
	}

	// Non-VFS path: soft-delete via TrashFiles (os.Rename — microseconds on
	// any filesystem). Files disappear from listings immediately; real data
	// purge runs on the 30-day expiry sweep.
	if _, err := deps.StorageService().TrashFiles(storageutil.TrashFilesParams{
		RootDir:      rootDir,
		FilePaths:    filePaths,
		DeviceSerial: serial,
	}); err != nil {
		return serverutil.InternalServerError(err)
	}

	// Return 202 before the (async) DB cleanup — the rename already hides the
	// files from all storage listings.
	go publishAndCleanup(deps, filePaths, serial)

	return serverutil.Accepted()
}

// publishAndCleanup fires delete events and removes DB records for each
// deleted path. Runs in a background goroutine for the non-VFS path so the
// HTTP response is not blocked on SQLite writes.
func publishAndCleanup(deps deputil.Dependencies, filePaths []string, serial string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, p := range filePaths {
		deps.EventBus().Publish(eventbus.Event{
			Kind:         eventbus.EventDelete,
			Path:         p,
			DeviceSerial: serial,
		})
		if database := deps.Database(); database != nil {
			if err := database.Queries.DeletePhotoFromAllAlbums(ctx, db.DeletePhotoFromAllAlbumsParams{
				DeviceSerial: serial,
				RelPath:      p,
			}); err != nil {
				log.Printf("autobutler: delete cleanup: remove album items for %q (serial=%q): %v", p, serial, err)
			}
			if err := database.Queries.DeletePhotoRotation(ctx, db.DeletePhotoRotationParams{
				DeviceSerial: serial,
				RelPath:      p,
			}); err != nil {
				log.Printf("autobutler: delete cleanup: remove rotation for %q (serial=%q): %v", p, serial, err)
			}
		}
	}
}

var deleteFilesRoute = serverutil.ApiRoute(
	"DELETE", "/cirrus", deleteFiles,
)
