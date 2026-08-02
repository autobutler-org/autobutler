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
// @Description Soft-delete (trash) files under the specified root directory.
// @Description Files are moved to .trash via os.Rename — a pure metadata op —
// @Description so the response returns immediately. DB cleanup and event
// @Description publishing happen in a background goroutine.
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

	// ── Phase 1: fast rename to trash ────────────────────────────────────────
	//
	// Use VFS.Trash when no serial and a VFS is registered (routes through
	// StorageServiceVFS which in turn calls TrashFiles). Fall back to the
	// StorageService.TrashFiles path for direct-serial requests.
	//
	// os.Rename is a pure metadata operation — microseconds on any filesystem,
	// including the Pi's SD card. This is what makes the response near-instant
	// for large batches.
	usedVFS := false
	if serial == "" {
		if reg := deps.VFSRegistry(); reg != nil {
			if fsys, ok := reg.Get("files"); ok {
				for _, p := range filePaths {
					if err := fsys.Trash(c.Request.Context(), p, vfs.TrashOptions{Recursive: true}); err != nil && err != vfs.ErrNotFound {
						return serverutil.InternalServerError(err)
					}
				}
				usedVFS = true
			}
		}
	}
	if !usedVFS {
		if _, err := deps.StorageService().TrashFiles(storageutil.TrashFilesParams{
			RootDir:      rootDir,
			FilePaths:    filePaths,
			DeviceSerial: serial,
		}); err != nil {
			return serverutil.InternalServerError(err)
		}
	}

	// ── Phase 2: background cleanup ───────────────────────────────────────────
	//
	// DB writes and event publishing are decoupled from the HTTP response.
	// The files are already invisible to directory listings at this point
	// (they live in .trash), so the client can update its UI immediately on 202.
	//
	// Capture all values by copy — the gin.Context must not be accessed after
	// the handler returns.
	database := deps.Database()
	bus := deps.EventBus()
	go func(paths []string, deviceSerial string, database *db.DatabaseSqlc, bus *eventbus.Bus) {
		for _, p := range paths {
			bus.Publish(eventbus.Event{
				Kind:         eventbus.EventDelete,
				Path:         p,
				DeviceSerial: deviceSerial,
			})
			if database == nil {
				continue
			}
			if err := database.Queries.DeletePhotoFromAllAlbums(context.Background(), db.DeletePhotoFromAllAlbumsParams{
				DeviceSerial: deviceSerial,
				RelPath:      p,
			}); err != nil {
				log.Printf("autobutler: delete cleanup: remove album items for %q (serial=%q): %v", p, deviceSerial, err)
			}
			if err := database.Queries.DeletePhotoRotation(context.Background(), db.DeletePhotoRotationParams{
				DeviceSerial: deviceSerial,
				RelPath:      p,
			}); err != nil {
				log.Printf("autobutler: delete cleanup: remove rotation for %q (serial=%q): %v", p, deviceSerial, err)
			}
		}
	}(filePaths, serial, database, bus)

	return serverutil.Ok()
}

var deleteFilesRoute = serverutil.ApiRoute(
	"DELETE", "/cirrus", deleteFiles,
)
