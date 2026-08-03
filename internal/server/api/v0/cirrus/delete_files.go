package v0_files

import (
	"context"
	"errors"
	"log"
	"sync"

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

	// Phase 1: fast rename — returns to client immediately after this.
	usedVFS := false
	if serial == "" {
		if reg := deps.VFSRegistry(); reg != nil {
			if fsys, ok := reg.Get("files"); ok {
				// VFS path: parallelise the renames with a bounded worker pool.
				if err := parallelVFSDelete(c.Request.Context(), fsys, filePaths); err != nil {
					return serverutil.InternalServerError(err)
				}
				usedVFS = true
			}
		}
	}
	if !usedVFS {
		// Non-VFS path: use TrashFiles (os.Rename — microseconds per file).
		if _, err := deps.StorageService().TrashFiles(storageutil.TrashFilesParams{
			RootDir:      rootDir,
			FilePaths:    filePaths,
			DeviceSerial: serial,
		}); err != nil {
			return serverutil.InternalServerError(err)
		}
	}

	// Phase 2: DB cleanup + event publishing — fire-and-forget in background.
	// Files are already gone from the listing; the client doesn't need to wait.
	database := deps.Database()
	bus := deps.EventBus()
	go func() {
		for _, p := range filePaths {
			bus.Publish(eventbus.Event{
				Kind:         eventbus.EventDelete,
				Path:         p,
				DeviceSerial: serial,
			})
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
	}()

	return serverutil.Ok()
}

// parallelVFSDelete deletes files via the VFS with up to 8 concurrent workers.
func parallelVFSDelete(ctx context.Context, fsys vfs.VFS, filePaths []string) error {
	const maxWorkers = 8
	sem := make(chan struct{}, maxWorkers)

	var (
		mu       sync.Mutex
		firstErr error
	)

	var wg sync.WaitGroup
	for _, p := range filePaths {
		p := p
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if err := fsys.Delete(ctx, p, vfs.DeleteOptions{Recursive: true}); err != nil && err != vfs.ErrNotFound {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return firstErr
}

var deleteFilesRoute = serverutil.ApiRoute(
	"DELETE", "/cirrus", deleteFiles,
)
