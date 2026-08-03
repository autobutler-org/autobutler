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

	"github.com/gin-gonic/gin"
)

// deleteFiles godoc
// @Summary Delete files
// @Description Move files to the trash for deferred deletion. Returns 202 immediately after
// @Description the rename so the caller does not block on filesystem I/O or DB cleanup.
// @Description DB record removal and event publishing happen asynchronously in a background goroutine.
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

	// Move files to trash with a single os.Rename per file — pure metadata op,
	// near-instant even on SD card. This replaces the old serial os.RemoveAll
	// (slow on SD card due to filesystem journal flushes) and the VFS.Delete
	// path (also serial). Both old paths blocked the request goroutine for the
	// full duration of the filesystem work.
	if _, err := deps.StorageService().TrashFiles(storageutil.TrashFilesParams{
		RootDir:      rootDir,
		FilePaths:    filePaths,
		DeviceSerial: serial,
	}); err != nil {
		return serverutil.InternalServerError(err)
	}

	// DB cleanup and event publishing run in the background so the 202 is
	// returned to the client before any SQLite writes happen. For 45 files this
	// saves 90 sequential DB round-trips from the request path.
	go func() {
		for _, p := range filePaths {
			deps.EventBus().Publish(eventbus.Event{
				Kind:         eventbus.EventDelete,
				Path:         p,
				DeviceSerial: serial,
			})
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
	}()

	return serverutil.Ok()
}

var deleteFilesRoute = serverutil.ApiRoute(
	"DELETE", "/cirrus", deleteFiles,
)
