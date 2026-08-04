// Package searchutil provides full-text search over AutoButler document files.
//
// Supported formats:
//   - .abdoc  — Quill Delta JSON: {"ops": [{insert: "..."}, ...]}
//   - .absheet — AutoButler sheet JSON: {"tabs": [{data: {rows: [[...]]}}]}
//   - .txt, .md and other text/* files — raw bytes
//
// The extracted plain text is stored in the SQLite FTS5 virtual table
// document_fts (see migration 014_document_fts).
package searchutil

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

// IsIndexable returns true when filename has an extension we can extract text from.
// Use this for fast extension-only checks before reading file data.
func IsIndexable(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".abdoc", ".absheet",
		".txt", ".md", ".csv", ".json", ".yaml", ".yml",
		".xml", ".html", ".htm", ".css", ".js", ".ts",
		".go", ".py", ".rs", ".sh":
		return true
	}
	return false
}

// ExtractText returns plain text content suitable for FTS indexing.
// Returns ("", false) when the file type is not indexable.
func ExtractText(filename string, data []byte) (string, bool) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".abdoc":
		return extractAbdoc(data)
	case ".absheet":
		return extractAbsheet(data)
	case ".txt", ".md", ".csv", ".json", ".yaml", ".yml",
		".xml", ".html", ".htm", ".css", ".js", ".ts",
		".go", ".py", ".rs", ".sh":
		return string(data), true
	default:
		return "", false
	}
}

// extractAbdoc pulls plain text from a Quill Delta JSON file.
// Format: {"ops": [{"insert": "text"}, {"insert": {"image": ...}}, ...]}
// Only string inserts contain searchable text; embed objects are skipped.
func extractAbdoc(data []byte) (string, bool) {
	var doc struct {
		Ops []json.RawMessage `json:"ops"`
	}
	if err := json.Unmarshal(data, &doc); err != nil || len(doc.Ops) == 0 {
		return "", false
	}

	var sb strings.Builder
	for _, raw := range doc.Ops {
		// Each op is {"insert": <string|object>, "attributes": {...}}
		var op struct {
			Insert json.RawMessage `json:"insert"`
		}
		if err := json.Unmarshal(raw, &op); err != nil || op.Insert == nil {
			continue
		}
		// String insert: "hello world\n"
		var s string
		if err := json.Unmarshal(op.Insert, &s); err == nil {
			sb.WriteString(s)
		}
		// Object insert (image, video, etc.): skip
	}
	text := strings.TrimSpace(sb.String())
	return text, text != ""
}

// extractAbsheet pulls plain text from an AutoButler spreadsheet JSON file.
// Format: {"tabs": [{"name": "Sheet1", "data": {"rows": [["a","b"],["c","d"]]}}]}
func extractAbsheet(data []byte) (string, bool) {
	var sheet struct {
		Tabs []struct {
			Name string `json:"name"`
			Data struct {
				Rows [][]json.RawMessage `json:"rows"`
			} `json:"data"`
		} `json:"tabs"`
	}
	if err := json.Unmarshal(data, &sheet); err != nil || len(sheet.Tabs) == 0 {
		return "", false
	}

	var sb strings.Builder
	for _, tab := range sheet.Tabs {
		if tab.Name != "" {
			sb.WriteString(tab.Name)
			sb.WriteByte('\n')
		}
		for _, row := range tab.Data.Rows {
			for i, cell := range row {
				var s string
				if err := json.Unmarshal(cell, &s); err == nil {
					if i > 0 {
						sb.WriteByte('\t')
					}
					sb.WriteString(s)
				}
			}
			sb.WriteByte('\n')
		}
	}
	text := strings.TrimSpace(sb.String())
	return text, text != ""
}
