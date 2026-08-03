package ftsutil

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/util/eventbus"
)

// StartEventSubscriber listens for file upload/delete events on the bus and
// keeps the FTS index up to date incrementally. It blocks until ctx is done
// and should be run in a goroutine.
//
// filesDir is the root of the primary cirrus directory (used to resolve
// relative paths for upload events).
func StartEventSubscriber(ctx context.Context, bus *eventbus.Bus, database *db.DatabaseSqlc, filesDir string) {
	ch, unsub := bus.Subscribe("fts-indexer")
	defer unsub()

	slog.Info("fts: event subscriber started")
	for {
		select {
		case <-ctx.Done():
			slog.Info("fts: event subscriber stopping")
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			handleEvent(ctx, evt, database, filesDir)
		}
	}
}

func handleEvent(ctx context.Context, evt eventbus.Event, database *db.DatabaseSqlc, filesDir string) {
	serial := evt.DeviceSerial
	path := evt.Path

	switch evt.Kind {
	case eventbus.EventUpload:
		if !IsIndexable(path) {
			return
		}
		fullPath := filepath.Join(filesDir, path)
		indexed, err := IndexFile(ctx, database.Queries, database.Db, serial, path, fullPath)
		if err != nil {
			slog.Warn("fts: index upload event", "path", path, "err", err)
		} else if indexed {
			slog.Debug("fts: indexed", "path", path)
		}

	case eventbus.EventDelete:
		if !IsIndexable(path) {
			return
		}
		if err := RemoveFile(ctx, database.Queries, database.Db, serial, path); err != nil {
			slog.Warn("fts: remove delete event", "path", path, "err", err)
		}

	case eventbus.EventMove:
		// Remove old path, index new path.
		if IsIndexable(evt.Path) {
			_ = RemoveFile(ctx, database.Queries, database.Db, serial, evt.Path)
		}
		if IsIndexable(evt.NewPath) {
			fullPath := filepath.Join(filesDir, evt.NewPath)
			_, _ = IndexFile(ctx, database.Queries, database.Db, serial, evt.NewPath, fullPath)
		}
	}
}
