package searchutil

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// SearchResult is a single FTS5 search hit.
type SearchResult struct {
	// Serial is the storage device serial that holds the file.
	Serial string `json:"serial"`
	// RelPath is the relative path of the file within the device's CirrusDir.
	RelPath string `json:"relPath"`
	// Snippet is a highlighted excerpt from the matched content.
	// Matched terms are wrapped in <b>…</b>. The caller is responsible for
	// sanitising this before rendering in HTML (the excerpt is derived from
	// file contents, not user input, but treat it with care).
	Snippet string `json:"snippet"`
}

// DefaultLimit is the maximum number of results returned by Search when the
// caller does not specify a limit.
const DefaultLimit = 50

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

// Search runs a full-text query against the FTS5 index and returns up to
// limit results ordered by relevance rank. If limit <= 0, DefaultLimit is used.
// The query string is passed directly to FTS5's MATCH operator — callers
// should sanitise it for user-facing inputs (e.g. quote terms to avoid FTS5
// syntax errors).
func Search(ctx context.Context, db *sql.DB, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = DefaultLimit
	}
	// Sanitise the query: wrap in double-quotes if it contains no FTS5
	// operators so a bare word search never triggers syntax errors.
	safeQuery := sanitiseFTSQuery(query)

	rows, err := db.QueryContext(ctx, `
		SELECT
		    fc.serial,
		    fc.rel_path,
		    snippet(file_content_fts, 0, '<b>', '</b>', '…', 20) AS snippet
		FROM file_content_fts
		JOIN file_content fc ON fc.id = file_content_fts.rowid
		WHERE file_content_fts MATCH ?
		ORDER BY rank
		LIMIT ?`, safeQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("fts5 search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.Serial, &r.RelPath, &r.Snippet); err != nil {
			return nil, fmt.Errorf("scan search result: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("fts5 result iteration: %w", err)
	}
	if results == nil {
		results = []SearchResult{}
	}
	return results, nil
}

// sanitiseFTSQuery wraps the query in double-quotes if it contains no FTS5
// operators, preventing syntax errors from bare special characters.
// FTS5 operators: AND, OR, NOT, NEAR, column filters, prefix wildcards.
func sanitiseFTSQuery(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return `""`
	}
	// If the query already contains FTS5 operators or quotes, pass it through.
	upper := strings.ToUpper(q)
	if strings.ContainsAny(q, `"*:()`) ||
		strings.Contains(upper, " AND ") ||
		strings.Contains(upper, " OR ") ||
		strings.Contains(upper, " NOT ") ||
		strings.Contains(upper, "NEAR(") {
		return q
	}
	// Plain term(s) — wrap in double-quotes to treat as phrase search.
	return `"` + strings.ReplaceAll(q, `"`, `""`) + `"`
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
