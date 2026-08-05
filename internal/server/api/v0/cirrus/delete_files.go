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
		// Use TrashFiles (os.Rename) instead of DeleteFiles (os.RemoveAll).
		// Rename is a pure metadata operation — microseconds even on an SD card —
		// so we can return 202 immediately and let the real data purge happen on
		// the 30-day expiry sweep. Files are invisible to listings as soon as
		// the rename lands.
		if _, err := deps.StorageService().TrashFiles(storageutil.TrashFilesParams{
			RootDir:      rootDir,
			FilePaths:    filePaths,
			DeviceSerial: serial,
		}); err != nil {
			return serverutil.InternalServerError(err)
		}
	}

	// Fire DB cleanup and event publishing asynchronously. The files are already
	// invisible (renamed to .trash or deleted via VFS), so the client doesn't
	// need to wait for the DB writes to return.
	go publishAndCleanup(deps, filePaths, serial)

	return serverutil.Ok()
}

// publishAndCleanup publishes delete events and removes per-file DB records.
// Runs in a goroutine so the HTTP handler returns immediately after the
// filesystem operation completes.
func publishAndCleanup(deps deputil.Dependencies, filePaths []string, serial string) {
	for _, p := range filePaths {
		deps.EventBus().Publish(eventbus.Event{
			Kind:         eventbus.EventDelete,
			Path:         p,
			DeviceSerial: serial,
		})
		database := deps.Database()
		if database == nil {
			continue
		}
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

var deleteFilesRoute = serverutil.ApiRoute(
	"DELETE", "/cirrus", deleteFiles,
)
