// Package searchutil provides text extraction and full-text search over
// indexed file contents using SQLite FTS5.
//
// Supported for text extraction (in order of fidelity):
//   - Plaintext files (.txt, .md, .csv, .log, .yaml, .yml, .toml, .json, .xml,
//     .html, .htm, .ini, .cfg, .conf, .sh, .py, .go, .js, .ts, .css, .sql)
//   - All other files: content is not indexed (empty string returned)
//
// EPUB and PDF extraction is not yet implemented — tracked in #1339. Adding
// them only requires extending ExtractText without any schema changes.
package searchutil

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// MaxExtractBytes is the maximum number of bytes read from a file for indexing.
// Files larger than this are truncated at a UTF-8 boundary to avoid storing
// enormous documents in the search index.
const MaxExtractBytes = 512 * 1024 // 512 KB

// extractableExtensions is the set of file extensions whose contents are safe
// to read as UTF-8 text for indexing. Binary formats (images, video, audio,
// executables) are excluded.
var extractableExtensions = map[string]bool{
	".txt":  true,
	".md":   true,
	".csv":  true,
	".log":  true,
	".yaml": true,
	".yml":  true,
	".toml": true,
	".json": true,
	".xml":  true,
	".html": true,
	".htm":  true,
	".ini":  true,
	".cfg":  true,
	".conf": true,
	".sh":   true,
	".py":   true,
	".go":   true,
	".js":   true,
	".ts":   true,
	".css":  true,
	".sql":  true,
	".rst":  true,
	".tex":  true,
}

// IsIndexable reports whether the file at path is eligible for content
// extraction based on its extension. It does not read the file.
func IsIndexable(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return extractableExtensions[ext]
}

// ExtractText reads the file at path and returns its contents as a UTF-8
// string suitable for FTS5 indexing. At most MaxExtractBytes are read.
// If the file cannot be opened, is not indexable, or its contents are not
// valid UTF-8, an empty string is returned (no error — non-indexable files
// are silently skipped to keep the indexer simple).
func ExtractText(path string) string {
	if !IsIndexable(path) {
		return ""
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	return extractReader(f)
}

// extractReader reads up to MaxExtractBytes from r, truncating at a valid
// UTF-8 boundary. Non-UTF-8 content returns an empty string.
func extractReader(r io.Reader) string {
	buf := make([]byte, MaxExtractBytes)
	n, err := io.ReadFull(r, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return ""
	}
	data := buf[:n]
	if !utf8.Valid(data) {
		// Trim to last valid rune boundary.
		for !utf8.Valid(data) && len(data) > 0 {
			data = data[:len(data)-1]
		}
	}
	if !utf8.Valid(data) {
		return ""
	}
	return string(data)
}
