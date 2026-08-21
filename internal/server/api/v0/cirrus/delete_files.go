package v0_files

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/eventbus"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
	"github.com/autobutler-org/quark/pkg/vfs"

	"github.com/gin-gonic/gin"
)

// deleteFiles godoc
// @Summary Delete files
// @Description Soft-delete files via rename to trash, returning immediately. DB cleanup and events are dispatched in the background.
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

	// ── Phase 1: fast filesystem op (returns in < 1 s even for large batches) ─

	usedVFS := false
	if serial == "" {
		if reg := deps.VFSRegistry(); reg != nil {
			if fsys, fsysOK := reg.Get("files"); fsysOK {
				// Parallel VFS deletes — each call is independent.
				var wg sync.WaitGroup
				errs := make([]error, len(filePaths))
				ctx := c.Request.Context()
				for i, p := range filePaths {
					wg.Add(1)
					go func(i int, p string) {
						defer wg.Done()
						if err := fsys.Delete(ctx, p, vfs.DeleteOptions{Recursive: true}); err != nil && err != vfs.ErrNotFound {
							errs[i] = err
						}
					}(i, p)
				}
				wg.Wait()
				for _, err := range errs {
					if err != nil {
						return serverutil.InternalServerError(err)
					}
				}
				usedVFS = true
			}
		}
	}

	if !usedVFS {
		if serial != "" {
			// Fast path: rename to .trash/ — metadata-only op, microseconds on SD card.
			if _, err := deps.StorageService().TrashFiles(storageutil.TrashFilesParams{
				RootDir:      rootDir,
				FilePaths:    filePaths,
				DeviceSerial: serial,
			}); err != nil {
				return serverutil.InternalServerError(err)
			}
		} else {
			// Fallback for unknown-serial callers (rare; VFS covers most of these).
			if _, err := deps.StorageService().DeleteFiles(storageutil.DeleteFilesParams{
				RootDir:      rootDir,
				FilePaths:    filePaths,
				DeviceSerial: serial,
			}); err != nil {
				return serverutil.InternalServerError(err)
			}
		}
	}

	// ── Phase 2: async cleanup — event bus + DB ──────────────────────────────
	// Capture everything the goroutine needs before the context is cancelled.
	bus := deps.EventBus()
	database := deps.Database()
	pathsCopy := append([]string(nil), filePaths...)

	go func() {
		for _, p := range pathsCopy {
			bus.Publish(eventbus.Event{
				Kind:         eventbus.EventDelete,
				Path:         p,
				DeviceSerial: serial,
			})
			if database == nil {
				continue
			}
			ctx := context.Background()
			if err := database.Queries.DeletePhotoFromAllAlbums(ctx, db.DeletePhotoFromAllAlbumsParams{
				DeviceSerial: serial,
				RelPath:      p,
			}); err != nil {
				log.Printf("quark: delete cleanup: remove album items for %q (serial=%q): %v", p, serial, err)
			}
			if err := database.Queries.DeletePhotoRotation(ctx, db.DeletePhotoRotationParams{
				DeviceSerial: serial,
				RelPath:      p,
			}); err != nil {
				log.Printf("quark: delete cleanup: remove rotation for %q (serial=%q): %v", p, serial, err)
			}
		}
	}()

	return serverutil.Ok()
}

var deleteFilesRoute = serverutil.ApiRoute(
	"DELETE", "/cirrus", deleteFiles,
)
