package ftsutil

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/autobutler-org/autobutler/internal/db"
)

// IndexFile extracts text from a file and upserts it into the FTS index.
// If the file content hash hasn't changed since the last index, it is a no-op.
// Returns (indexed, error) where indexed=false means the file was skipped.
func IndexFile(ctx context.Context, queries *db.Queries, database *sql.DB, deviceSerial, relPath, fullPath string) (bool, error) {
	if !IsIndexable(relPath) {
		return false, nil
	}

	text, hash, err := ExtractText(ctx, fullPath)
	if err != nil {
		return false, fmt.Errorf("extract text from %s: %w", relPath, err)
	}
	if text == "" {
		return false, nil
	}

	// Check if the hash changed to avoid redundant index updates.
	existing, err := queries.GetFTSDocument(ctx, db.GetFTSDocumentParams{
		DeviceSerial: deviceSerial,
		RelPath:      relPath,
	})
	if err == nil && existing.ContentHash == hash {
		return false, nil // unchanged
	}

	// Upsert fts_documents row.
	if err := queries.UpsertFTSDocument(ctx, db.UpsertFTSDocumentParams{
		DeviceSerial: deviceSerial,
		RelPath:      relPath,
		ContentHash:  hash,
	}); err != nil {
		return false, fmt.Errorf("upsert fts_documents: %w", err)
	}

	// Get the row ID to update the FTS5 index.
	doc, err := queries.GetFTSDocument(ctx, db.GetFTSDocumentParams{
		DeviceSerial: deviceSerial,
		RelPath:      relPath,
	})
	if err != nil {
		return false, fmt.Errorf("get fts_document after upsert: %w", err)
	}

	// Update the FTS5 index directly (sqlc doesn't support FTS5 virtual table DDL).
	// First delete old entry, then insert new.
	_, _ = database.ExecContext(ctx,
		`INSERT INTO fts_index(fts_index, rowid, body) VALUES('delete', ?, ?)`,
		doc.ID, doc.ID,
	)
	if _, err := database.ExecContext(ctx,
		`INSERT INTO fts_index(rowid, body) VALUES(?, ?)`,
		doc.ID, text,
	); err != nil {
		return false, fmt.Errorf("insert fts_index: %w", err)
	}

	return true, nil
}

// RemoveFile removes a file from the FTS index.
func RemoveFile(ctx context.Context, queries *db.Queries, database *sql.DB, deviceSerial, relPath string) error {
	doc, err := queries.GetFTSDocument(ctx, db.GetFTSDocumentParams{
		DeviceSerial: deviceSerial,
		RelPath:      relPath,
	})
	if err != nil {
		return nil // already absent
	}
	_, _ = database.ExecContext(ctx,
		`INSERT INTO fts_index(fts_index, rowid, body) VALUES('delete', ?, ?)`,
		doc.ID, doc.ID,
	)
	return queries.DeleteFTSDocument(ctx, db.DeleteFTSDocumentParams{
		DeviceSerial: deviceSerial,
		RelPath:      relPath,
	})
}

// Search runs a full-text query and returns matching (deviceSerial, relPath) pairs.
// query is a raw FTS5 query string (e.g. "butler AND photos" or "\"exact phrase\"").
func Search(ctx context.Context, database *sql.DB, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := database.QueryContext(ctx,
		`SELECT d.device_serial, d.rel_path, rank
         FROM fts_index f
         JOIN fts_documents d ON d.id = f.rowid
         WHERE fts_index MATCH ?
         ORDER BY rank
         LIMIT ?`,
		query, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("fts query: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.DeviceSerial, &r.RelPath, &r.Rank); err != nil {
			slog.Warn("ftsutil: scan row", "err", err)
			continue
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// SearchResult is one hit from a full-text search.
type SearchResult struct {
	DeviceSerial string  `json:"deviceSerial,omitempty"`
	RelPath      string  `json:"relPath"`
	Rank         float64 `json:"rank"`
}
