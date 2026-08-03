// Package ftsutil provides text extraction and FTS5 indexing for the
// document full-text search feature.
//
// Supported formats:
//   - Plaintext (.txt, .md, .go, .py, .js, .ts, .html, .xml, .json, .yaml, .toml, .csv, .log, ...)
//   - EPUB (.epub) — extracts all chapter HTML and strips tags
//   - PDF (.pdf) — extracts text via pdftotext (poppler-utils), when available
//
// The index is kept in the SQLite fts_documents + fts_index tables.
// It is derived and rebuildable — callers treat it as a read-through cache.
package ftsutil

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// MaxExtractBytes is the maximum number of text bytes extracted per file.
// Guards against indexing enormous logs or generated files.
const MaxExtractBytes = 2 * 1024 * 1024 // 2 MiB of extracted text

// IndexableExtensions is the set of file extensions eligible for FTS indexing.
var IndexableExtensions = map[string]struct{}{
	".txt": {}, ".md": {}, ".markdown": {},
	".go": {}, ".py": {}, ".js": {}, ".ts": {}, ".jsx": {}, ".tsx": {},
	".html": {}, ".htm": {}, ".xml": {}, ".svg": {},
	".json": {}, ".yaml": {}, ".yml": {}, ".toml": {}, ".ini": {}, ".env": {},
	".csv": {}, ".tsv": {}, ".log": {},
	".sh": {}, ".bash": {}, ".zsh": {},
	".c": {}, ".cpp": {}, ".h": {}, ".rs": {}, ".java": {}, ".kt": {},
	".rb": {}, ".php": {}, ".swift": {}, ".dart": {},
	".sql": {}, ".conf": {}, ".cfg": {},
	".epub": {},
	".pdf":  {},
}

// IsIndexable reports whether the given filename should be indexed.
func IsIndexable(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	_, ok := IndexableExtensions[ext]
	return ok
}

// PdfToTextAvailable reports whether pdftotext (poppler-utils) is installed.
func PdfToTextAvailable() bool {
	_, err := exec.LookPath("pdftotext")
	return err == nil
}

// ExtractText extracts readable text from a file. Returns (text, contentHash, error).
// contentHash is the SHA-256 of the extracted text (not the file) for incremental updates.
func ExtractText(ctx context.Context, fullPath string) (string, string, error) {
	ext := strings.ToLower(filepath.Ext(fullPath))
	var text string
	var err error

	switch ext {
	case ".epub":
		text, err = extractEPUB(fullPath)
	case ".pdf":
		text, err = extractPDF(ctx, fullPath)
	default:
		text, err = extractPlaintext(fullPath)
	}
	if err != nil {
		return "", "", err
	}

	if len(text) > MaxExtractBytes {
		text = text[:MaxExtractBytes]
	}

	h := sha256.Sum256([]byte(text))
	hash := hex.EncodeToString(h[:])
	return text, hash, nil
}

// extractPlaintext reads a text file, normalising line endings.
func extractPlaintext(fullPath string) (string, error) {
	f, err := os.Open(fullPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var sb strings.Builder
	sb.Grow(64 * 1024)
	scanner := bufio.NewScanner(io.LimitReader(f, MaxExtractBytes))
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		sb.WriteString(scanner.Text())
		sb.WriteByte('\n')
	}
	return sb.String(), scanner.Err()
}

// extractEPUB extracts text content from an EPUB file (a ZIP of HTML chapters).
func extractEPUB(fullPath string) (string, error) {
	r, err := zip.OpenReader(fullPath)
	if err != nil {
		return "", fmt.Errorf("open epub: %w", err)
	}
	defer r.Close()

	var sb strings.Builder
	for _, f := range r.File {
		if sb.Len() >= MaxExtractBytes {
			break
		}
		ext := strings.ToLower(filepath.Ext(f.Name))
		if ext != ".html" && ext != ".htm" && ext != ".xhtml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			slog.Warn("ftsutil: epub: open chapter", "file", f.Name, "err", err)
			continue
		}
		data, err := io.ReadAll(io.LimitReader(rc, int64(MaxExtractBytes)))
		rc.Close()
		if err != nil {
			continue
		}
		sb.WriteString(stripHTML(string(data)))
		sb.WriteByte('\n')
	}
	return sb.String(), nil
}

// extractPDF extracts text from a PDF using pdftotext (poppler-utils).
// Falls back to empty string with no error when pdftotext is not installed.
func extractPDF(ctx context.Context, fullPath string) (string, error) {
	if !PdfToTextAvailable() {
		return "", nil
	}
	out, err := exec.CommandContext(ctx, "pdftotext", fullPath, "-").Output()
	if err != nil {
		// pdftotext exits non-zero for encrypted PDFs etc. — not fatal.
		slog.Warn("ftsutil: pdftotext", "file", fullPath, "err", err)
		return "", nil
	}
	return string(out), nil
}

// stripHTML removes HTML/XML tags from a string, leaving whitespace-separated words.
func stripHTML(s string) string {
	var buf bytes.Buffer
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
			buf.WriteByte(' ')
		case r == '>':
			inTag = false
		case !inTag:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}
