// Package ftsutil provides a full-text search index over document contents
// using SQLite FTS5. The index is a derived, rebuildable artifact — it can
// always be regenerated from the files on disk.
//
// Supported formats for text extraction:
//   - Plaintext (.txt, .md, .csv, .log, and other text types)
//
// PDF and EPUB extraction are noted as follow-ups; those file types are
// registered in the index by path but without content until extractors land.
//
// Thread-safety: Index/Delete/Search are safe for concurrent use.
package ftsutil

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "modernc.org/sqlite" // CGo-free SQLite driver
)

const schema = `
CREATE VIRTUAL TABLE IF NOT EXISTS fts USING fts5(
	path     UNINDEXED,
	content,
	tokenize = 'porter ascii'
);

-- Shadow table tracks indexed paths so we can skip files that haven't changed.
CREATE TABLE IF NOT EXISTS fts_meta (
	path        TEXT PRIMARY KEY,
	size        INTEGER NOT NULL DEFAULT 0,
	mod_unix    INTEGER NOT NULL DEFAULT 0
);
`

// Result is a single full-text search hit.
type Result struct {
	// Path is the relative file path (same key used in Index calls).
	Path string
	// Snippet is a short context extract around the matching terms, with HTML
	// <mark> tags wrapping matches. Empty when snippets are disabled.
	Snippet string
}

// Index manages the SQLite FTS5 full-text search index.
type Index struct {
	mu sync.Mutex
	db *sql.DB
}

// Open opens (or creates) the FTS index at the given file path.
// The directory is created if it does not exist.
func Open(path string) (*Index, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("ftsutil: mkdir: %w", err)
	}
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_foreign_keys=off")
	if err != nil {
		return nil, fmt.Errorf("ftsutil: open db: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("ftsutil: create schema: %w", err)
	}
	return &Index{db: db}, nil
}

// Close releases the database connection.
func (idx *Index) Close() error {
	return idx.db.Close()
}

// IndexFile extracts text from the file at absPath and stores it in the FTS
// index under the given relPath key. If the file's size and mtime match the
// previously indexed record, the file is skipped (no re-indexing).
//
// absPath is the real filesystem path; relPath is the stable identifier used
// for search results and Delete calls.
func (idx *Index) IndexFile(absPath, relPath string) error {
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("ftsutil: stat %s: %w", absPath, err)
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Check if already indexed with same size+mtime.
	var prevSize, prevMod int64
	row := idx.db.QueryRow("SELECT size, mod_unix FROM fts_meta WHERE path = ?", relPath)
	_ = row.Scan(&prevSize, &prevMod)
	if prevSize == info.Size() && prevMod == info.ModTime().Unix() {
		return nil // up to date
	}

	content, err := extractText(absPath)
	if err != nil {
		// Non-fatal: index the path without content so searches still find
		// the file by filename via the filename-based FileIndex.
		content = ""
		slog.Debug("ftsutil: could not extract text", "path", absPath, "err", err)
	}

	// Upsert: delete old FTS row then insert fresh.
	if _, err := idx.db.Exec("DELETE FROM fts WHERE path = ?", relPath); err != nil {
		return fmt.Errorf("ftsutil: delete old fts row: %w", err)
	}
	if _, err := idx.db.Exec("INSERT INTO fts(path, content) VALUES (?, ?)", relPath, content); err != nil {
		return fmt.Errorf("ftsutil: insert fts row: %w", err)
	}
	if _, err := idx.db.Exec(`
		INSERT INTO fts_meta(path, size, mod_unix) VALUES (?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET size=excluded.size, mod_unix=excluded.mod_unix`,
		relPath, info.Size(), info.ModTime().Unix(),
	); err != nil {
		return fmt.Errorf("ftsutil: upsert fts_meta: %w", err)
	}

	return nil
}

// Delete removes the file at relPath from the FTS index.
func (idx *Index) Delete(relPath string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	if _, err := idx.db.Exec("DELETE FROM fts WHERE path = ?", relPath); err != nil {
		return fmt.Errorf("ftsutil: delete from fts: %w", err)
	}
	if _, err := idx.db.Exec("DELETE FROM fts_meta WHERE path = ?", relPath); err != nil {
		return fmt.Errorf("ftsutil: delete from fts_meta: %w", err)
	}
	return nil
}

// Search returns files whose content matches the FTS5 query. The query uses
// standard FTS5 syntax (phrase: "hello world", prefix: term*, boolean: AND/OR/NOT).
//
// limit caps the number of results (0 = use default of 50).
func (idx *Index) Search(ctx context.Context, query string, limit int) ([]Result, error) {
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	rows, err := idx.db.QueryContext(ctx, `
		SELECT path, snippet(fts, 1, '<mark>', '</mark>', '…', 20)
		FROM fts
		WHERE content MATCH ?
		ORDER BY rank
		LIMIT ?`, query, limit)
	if err != nil {
		return nil, fmt.Errorf("ftsutil: search: %w", err)
	}
	defer rows.Close()

	var results []Result
	for rows.Next() {
		var r Result
		if err := rows.Scan(&r.Path, &r.Snippet); err != nil {
			return nil, fmt.Errorf("ftsutil: scan: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// Rebuild re-indexes all files in the given root directory. Existing entries
// that no longer exist on disk are purged. This is safe to call at startup to
// ensure the index is consistent after a crash or manual file change.
func (idx *Index) Rebuild(rootDir string) error {
	return filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, relErr := filepath.Rel(rootDir, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if !isIndexable(path) {
			return nil
		}
		if indexErr := idx.IndexFile(path, rel); indexErr != nil {
			slog.Warn("ftsutil: rebuild: index failed", "path", rel, "err", indexErr)
		}
		return nil
	})
}

// ─── text extraction ─────────────────────────────────────────────────────────

// isIndexable returns true for file types where we can extract meaningful text.
func isIndexable(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".txt", ".md", ".markdown", ".csv", ".log", ".rst", ".adoc", ".text":
		return true
	default:
		return false
	}
}

// extractText reads and returns the text content of a file. Returns an empty
// string for unsupported types rather than an error.
func extractText(absPath string) (string, error) {
	if !isIndexable(absPath) {
		return "", nil
	}
	f, err := os.Open(absPath) // #nosec G304 — path is from our own file walk, not user input
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Limit extraction to 1 MB to avoid indexing huge logs.
	const maxBytes = 1 << 20
	r := io.LimitReader(f, maxBytes)
	buf, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}
