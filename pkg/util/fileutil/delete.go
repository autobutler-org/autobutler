package fileutil

import (
	"context"
	"log"
	"sync"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/eventbus"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
	"github.com/autobutler-org/quark/pkg/vfs"
)

// DeleteFilesParams soft-deletes a batch of files. The filesystem half returns
// in under a second even for a large batch; the database and event-bus cleanup
// it leaves behind is dispatched in the background.
type DeleteFilesParams struct {
	// Ctx bounds the VFS deletes. The background cleanup deliberately does not
	// use it: it outlives the request.
	Ctx context.Context
	// Registry deletes through the VFS when no serial routes past it.
	Registry vfs.Registry
	// Storage deletes for a device-scoped request, or when there is no VFS.
	Storage *storageutil.StorageService
	// EventBus is told about every deleted path.
	EventBus *eventbus.Bus
	// Database holds the album membership and rotation rows to clean up. Nil
	// skips that half.
	Database *db.DatabaseSqlc
	// RootDir is the directory the paths are relative to.
	RootDir string
	// FilePaths are the files to delete.
	FilePaths []string
	// Serial identifies the device, empty for the internal one.
	Serial string
}

// DeleteFilesResult reports a completed delete. The background cleanup it
// started may still be running.
type DeleteFilesResult struct{}

// DeleteFiles deletes files and starts the cleanup their absence implies.
func DeleteFiles(params DeleteFilesParams) (DeleteFilesResult, error) {
	// ── Phase 1: fast filesystem op (returns in < 1 s even for large batches) ─

	usedVFS := false
	if params.Serial == "" {
		if fsys := FilesVFS(params.Registry); fsys != nil {
			// Parallel VFS deletes — each call is independent.
			var wg sync.WaitGroup
			errs := make([]error, len(params.FilePaths))
			for i, p := range params.FilePaths {
				wg.Add(1)
				go func(i int, p string) {
					defer wg.Done()
					if err := fsys.Delete(params.Ctx, p, vfs.DeleteOptions{Recursive: true}); err != nil && err != vfs.ErrNotFound {
						errs[i] = err
					}
				}(i, p)
			}
			wg.Wait()
			for _, err := range errs {
				if err != nil {
					return DeleteFilesResult{}, err
				}
			}
			usedVFS = true
		}
	}

	if !usedVFS {
		if params.Serial != "" {
			// Fast path: rename to .trash/ — metadata-only op, microseconds on SD card.
			if _, err := params.Storage.TrashFiles(storageutil.TrashFilesParams{
				RootDir:      params.RootDir,
				FilePaths:    params.FilePaths,
				DeviceSerial: params.Serial,
			}); err != nil {
				return DeleteFilesResult{}, err
			}
		} else {
			// Fallback for unknown-serial callers (rare; VFS covers most of these).
			if _, err := params.Storage.DeleteFiles(storageutil.DeleteFilesParams{
				RootDir:      params.RootDir,
				FilePaths:    params.FilePaths,
				DeviceSerial: params.Serial,
			}); err != nil {
				return DeleteFilesResult{}, err
			}
		}
	}

	// ── Phase 2: async cleanup — event bus + DB ──────────────────────────────
	// Capture everything the goroutine needs before the context is cancelled.
	bus := params.EventBus
	database := params.Database
	serial := params.Serial
	pathsCopy := append([]string(nil), params.FilePaths...)

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

	return DeleteFilesResult{}, nil
}
