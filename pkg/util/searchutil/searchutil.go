// Package searchutil provides text extraction and full-text search over
// indexed file contents using SQLite FTS5.
//
// Supported for text extraction (in order of fidelity):
//   - Quark documents (.qdoc) and spreadsheets (.qsheet): the prose is
//     pulled out of the JSON envelope so the index holds readable text rather
//     than markup (see extractDelta and extractSheet)
//   - Plaintext files (.txt, .md, .csv, .log, .yaml, .yml, .toml, .json, .xml,
//     .html, .htm, .ini, .cfg, .conf, .sh, .py, .go, .js, .ts, .css, .sql)
//   - All other files: content is not indexed (empty string returned)
//
// EPUB and PDF extraction is not yet implemented — tracked in #1339. Adding
// them only requires extending ExtractText without any schema changes.
package searchutil

import (
	"context"
	"database/sql"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// MaxExtractBytes is the maximum number of bytes read from a file for indexing.
// Files larger than this are truncated at a UTF-8 boundary to avoid storing
// enormous documents in the search index.
const MaxExtractBytes = 512 * 1024 // 512 KB

// DefaultLimit is the maximum number of results returned by Search when the
// caller does not specify a limit.
const DefaultLimit = 50

// MaxLimit is the largest page a caller may ask for. A search that wants more
// hits than this wants a different query.
const MaxLimit = 200

// ParseLimit reads a requested result limit, falling back to DefaultLimit for
// anything missing or unparseable and clamping the page to MaxLimit.
func ParseLimit(raw string) int {
	limit := DefaultLimit
	if raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			if n > MaxLimit {
				n = MaxLimit
			}
			limit = n
		}
	}
	return limit
}

// BackfillResult reports what a BackfillTree pass did.
type BackfillResult struct {
	// Scanned is the number of files visited, indexable or not.
	Scanned int
	// Indexed is the number of files whose contents were written to the index.
	Indexed int
	// Failed is the number of files that could not be indexed. Individual
	// errors do not abort the walk.
	Failed int
}

// SearchResult is a single FTS5 search hit.
type SearchResult struct {
	// Serial is the storage device serial that holds the file.
	Serial string `json:"serial"`
	// RelPath is the relative path of the file within the device's FilesDir.
	RelPath string `json:"relPath"`
	// Snippet is a highlighted excerpt from the matched content.
	// Matched terms are wrapped in <b>…</b>. The caller is responsible for
	// sanitizing this before rendering in HTML (the excerpt is derived from
	// file contents, not user input, but treat it with care).
	Snippet string `json:"snippet"`
}

// IsIndexable reports whether the file at path is eligible for content
// extraction based on its extension. It does not read the file.
func IsIndexable(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return extractableExtensions[ext]
}

// UpsertContent indexes the extracted text for a file. If the file was
// previously indexed, its entry is updated. Callers should call ExtractText
// first and skip calling this function when the result is empty.
func UpsertContent(ctx context.Context, db *sql.DB, serial, relPath, extracted string) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO file_content (serial, rel_path, extracted, updated_at)
		 VALUES (?, ?, ?, datetime('now'))
		 ON CONFLICT(serial, rel_path) DO UPDATE SET
		     extracted  = excluded.extracted,
		     updated_at = excluded.updated_at`,
		serial, relPath, extracted)
	return err
}

// DeleteContent removes the index entry for a specific file.
func DeleteContent(ctx context.Context, db *sql.DB, serial, relPath string) error {
	_, err := db.ExecContext(ctx,
		`DELETE FROM file_content WHERE serial = ? AND rel_path = ?`,
		serial, relPath)
	return err
}

// DeleteContentBySerial removes all index entries for a storage device.
// Call this when a device is removed.
func DeleteContentBySerial(ctx context.Context, db *sql.DB, serial string) error {
	_, err := db.ExecContext(ctx,
		`DELETE FROM file_content WHERE serial = ?`,
		serial)
	return err
}

// IndexFile extracts text from path and upserts it into the index.
// If the file is not indexable (non-text format), the call is a no-op.
func IndexFile(ctx context.Context, db *sql.DB, serial, relPath, absPath string) error {
	text := ExtractText(absPath)
	if text == "" {
		return nil
	}
	return UpsertContent(ctx, db, serial, relPath, text)
}

// IndexFileWithTimeout is like IndexFile but cancels extraction after d.
func IndexFileWithTimeout(db *sql.DB, serial, relPath, absPath string, d time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	return IndexFile(ctx, db, serial, relPath, absPath)
}
