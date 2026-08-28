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
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
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
	// Quark's own document formats. Both are JSON envelopes, so they are
	// routed through a structured extractor rather than indexed verbatim.
	".qdoc":   true,
	".qsheet": true,

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
	raw := extractReader(f)
	if raw == "" {
		return ""
	}

	// Quark's own formats wrap prose in JSON. Indexing the envelope
	// verbatim would work, but every snippet would be full of `{"ops":[{"insert":`
	// noise and queries would match on JSON keys, so pull the text out instead.
	// A document too large for MaxExtractBytes arrives here truncated and will
	// not parse; falling back to the raw text keeps it searchable.
	switch strings.ToLower(filepath.Ext(path)) {
	case ".qdoc":
		if text := extractDelta(raw); text != "" {
			return text
		}
	case ".qsheet":
		if text := extractSheet(raw); text != "" {
			return text
		}
	}
	return raw
}

// extractDelta pulls the prose out of a Quill Delta document, the format used
// by .qdoc files: {"ops":[{"insert":"some text"}, …]}. An op's insert is
// either a string (text) or an object (an embed such as an image); only
// strings carry indexable content. Returns "" when raw is not a Delta.
func extractDelta(raw string) string {
	var doc struct {
		Ops []struct {
			Insert any `json:"insert"`
		} `json:"ops"`
	}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return ""
	}
	var b strings.Builder
	for _, op := range doc.Ops {
		if s, ok := op.Insert.(string); ok {
			b.WriteString(s)
		}
	}
	return strings.TrimSpace(b.String())
}

// extractSheet pulls the tab names and cell values out of a spreadsheet, the
// format used by .qsheet files:
// {"tabs":[{"name":"Sheet 1","data":{"rows":[["a","b"], …]}}]}.
// Cells hold strings (including formulas like "=B1+B2", which stay indexed so
// they can be searched for) or numbers. Returns "" when raw is not a sheet.
func extractSheet(raw string) string {
	var doc struct {
		Tabs []struct {
			Name string `json:"name"`
			Data struct {
				Rows [][]any `json:"rows"`
			} `json:"data"`
		} `json:"tabs"`
	}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return ""
	}
	// Cells are joined by spaces so adjacent values never merge into one token.
	var parts []string
	for _, tab := range doc.Tabs {
		if tab.Name != "" {
			parts = append(parts, tab.Name)
		}
		for _, row := range tab.Data.Rows {
			for _, cell := range row {
				switch v := cell.(type) {
				case string:
					if v != "" {
						parts = append(parts, v)
					}
				case float64:
					parts = append(parts, strconv.FormatFloat(v, 'f', -1, 64))
				}
			}
		}
	}
	return strings.Join(parts, " ")
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
