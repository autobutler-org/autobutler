package searchutil

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/autobutler-org/autobutler/pkg/util/eventbus"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
)

// DocumentSearchResult is one hit returned by Search.
type DocumentSearchResult struct {
	DeviceSerial string `json:"deviceSerial"`
	RelPath      string `json:"relPath"`
	// Snippet is an HTML fragment with matched terms wrapped in <b>…</b>.
	Snippet string `json:"snippet"`
}

// Index manages the FTS5 document index in the given *sql.DB.
type Index struct {
	db *sql.DB
}

// NewIndex wraps db (which must have migration 014 applied).
func NewIndex(db *sql.DB) *Index {
	return &Index{db: db}
}

// Upsert indexes or re-indexes a single file. If the file is not indexable the
// row is silently deleted (to handle renames from an indexable to a non-indexable
// name). data must be the full file contents.
func (idx *Index) Upsert(ctx context.Context, deviceSerial, relPath string, data []byte) error {
	text, ok := ExtractText(filepath.Base(relPath), data)

	// Always delete first: ensures stale FTS rows don't accumulate.
	if _, err := idx.db.ExecContext(ctx,
		`DELETE FROM document_fts WHERE device_serial = ? AND rel_path = ?`,
		deviceSerial, relPath,
	); err != nil {
		return fmt.Errorf("searchutil: delete before upsert: %w", err)
	}

	if !ok || strings.TrimSpace(text) == "" {
		return nil // not indexable — deleted above, nothing to insert
	}

	if _, err := idx.db.ExecContext(ctx,
		`INSERT INTO document_fts (device_serial, rel_path, body) VALUES (?, ?, ?)`,
		deviceSerial, relPath, text,
	); err != nil {
		return fmt.Errorf("searchutil: insert FTS: %w", err)
	}
	return nil
}

// Delete removes the FTS entry for a file.
func (idx *Index) Delete(ctx context.Context, deviceSerial, relPath string) error {
	_, err := idx.db.ExecContext(ctx,
		`DELETE FROM document_fts WHERE device_serial = ? AND rel_path = ?`,
		deviceSerial, relPath,
	)
	return err
}

// Search returns up to 50 results matching query using FTS5 MATCH syntax.
// query may be a plain word, phrase ("exact phrase"), or boolean (a AND b).
func (idx *Index) Search(ctx context.Context, query string) ([]DocumentSearchResult, error) {
	// FTS5 MATCH syntax is already safe to pass through; the ? placeholder
	// prevents injection at the SQL level.
	rows, err := idx.db.QueryContext(ctx, `
		SELECT device_serial, rel_path,
		       snippet(document_fts, 2, '<b>', '</b>', '…', 16)
		FROM   document_fts
		WHERE  document_fts MATCH ?
		ORDER  BY rank
		LIMIT  50
	`, query)
	if err != nil {
		return nil, fmt.Errorf("searchutil: FTS query: %w", err)
	}
	defer rows.Close()

	var out []DocumentSearchResult
	for rows.Next() {
		var r DocumentSearchResult
		if err := rows.Scan(&r.DeviceSerial, &r.RelPath, &r.Snippet); err != nil {
			return nil, fmt.Errorf("searchutil: scan: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// RebuildFromDevices re-indexes all indexable files across managed devices.
// This is idempotent and can be run at startup to recover from a corrupt index.
func (idx *Index) RebuildFromDevices(ctx context.Context, storage *storageutil.StorageService) error {
	devices, err := storage.GetManagedDevices()
	if err != nil {
		return fmt.Errorf("searchutil: list devices: %w", err)
	}

	// Clear the index.
	if _, err := idx.db.ExecContext(ctx, `DELETE FROM document_fts`); err != nil {
		return fmt.Errorf("searchutil: clear FTS: %w", err)
	}

	for _, dev := range devices {
		serial := ""
		if dev.UsbInfo != nil {
			serial = dev.UsbInfo.GetSerial()
		}
		if err := idx.indexDevice(ctx, serial, dev.CirrusDir); err != nil {
			log.Printf("[search] rebuild: device %q: %v", serial, err)
		}
	}
	return nil
}

func (idx *Index) indexDevice(ctx context.Context, serial, cirrusDir string) error {
	return filepath.WalkDir(cirrusDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(cirrusDir, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)

		if !IsIndexable(d.Name()) {
			return nil // fast-path: skip non-indexable by name alone
		}

		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("[search] rebuild: read %q: %v", path, err)
			return nil
		}

		if err := idx.Upsert(ctx, serial, rel, data); err != nil {
			log.Printf("[search] rebuild: index %q: %v", rel, err)
		}
		return nil
	})
}

// HandleEvent updates the index in response to file events published on the EventBus.
func (idx *Index) HandleEvent(ctx context.Context, event eventbus.Event, cirrusDir string) {
	switch event.Kind {
	case eventbus.EventUpload:
		absPath := filepath.Join(cirrusDir, event.Path)
		data, err := os.ReadFile(absPath)
		if err != nil {
			log.Printf("[search] handle event: read %q: %v", event.Path, err)
			return
		}
		if err := idx.Upsert(ctx, event.DeviceSerial, event.Path, data); err != nil {
			log.Printf("[search] handle event: upsert %q: %v", event.Path, err)
		}
	case eventbus.EventDelete:
		if err := idx.Delete(ctx, event.DeviceSerial, event.Path); err != nil {
			log.Printf("[search] handle event: delete %q: %v", event.Path, err)
		}
	}
}
