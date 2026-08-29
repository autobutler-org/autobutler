package searchutil

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Search runs a full-text query against the FTS5 index and returns up to
// limit results ordered by relevance rank. If limit <= 0, DefaultLimit is used.
// The query string is passed directly to FTS5's MATCH operator — callers
// should sanitize it for user-facing inputs (e.g. quote terms to avoid FTS5
// syntax errors).
func Search(ctx context.Context, db *sql.DB, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = DefaultLimit
	}
	// Sanitize the query: wrap in double-quotes if it contains no FTS5
	// operators so a bare word search never triggers syntax errors.
	safeQuery := sanitizeFTSQuery(query)

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

// sanitizeFTSQuery wraps the query in double-quotes if it contains no FTS5
// operators, preventing syntax errors from bare special characters.
// FTS5 operators: AND, OR, NOT, NEAR, column filters, prefix wildcards.
func sanitizeFTSQuery(q string) string {
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
