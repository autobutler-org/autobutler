package migration

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ZipExtractor is a real implementation of ArchiveExtractor
type ZipExtractor struct{}

// NewZipExtractor creates a new zip extractor
func NewZipExtractor() *ZipExtractor {
	return &ZipExtractor{}
}

// Extract extracts a zip archive to a destination directory
func (e *ZipExtractor) Extract(ctx context.Context, archive io.Reader, destDir string) error {
	// Create temp file to store the archive (zip.Reader needs ReaderAt)
	tmpFile, err := os.CreateTemp("", "takeout-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Copy archive to temp file
	if _, err := io.Copy(tmpFile, archive); err != nil {
		return fmt.Errorf("failed to write archive: %w", err)
	}

	// Get file size
	stat, err := tmpFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat temp file: %w", err)
	}

	// Open zip reader
	zipReader, err := zip.NewReader(tmpFile, stat.Size())
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}

	// Extract all files
	for _, file := range zipReader.File {
		if err := extractFile(file, destDir); err != nil {
			return fmt.Errorf("failed to extract %s: %w", file.Name, err)
		}
	}

	return nil
}

// ListContents lists the contents of a zip archive without extracting
func (e *ZipExtractor) ListContents(ctx context.Context, archive io.Reader) ([]string, error) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "takeout-list-*.zip")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Copy archive to temp file
	if _, err := io.Copy(tmpFile, archive); err != nil {
		return nil, fmt.Errorf("failed to write archive: %w", err)
	}

	// Get file size
	stat, err := tmpFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat temp file: %w", err)
	}

	// Open zip reader
	zipReader, err := zip.NewReader(tmpFile, stat.Size())
	if err != nil {
		return nil, fmt.Errorf("failed to open zip: %w", err)
	}

	// List all files
	contents := make([]string, 0, len(zipReader.File))
	for _, file := range zipReader.File {
		contents = append(contents, file.Name)
	}

	return contents, nil
}

func extractFile(file *zip.File, destDir string) error {
	// Clean the file path to prevent directory traversal attacks
	filePath := filepath.Join(destDir, file.Name)
	if !strings.HasPrefix(filePath, filepath.Clean(destDir)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", file.Name)
	}

	// Create directory if it's a directory entry
	if file.FileInfo().IsDir() {
		return os.MkdirAll(filePath, os.ModePerm)
	}

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Open source file
	srcFile, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer srcFile.Close()

	// Create destination file
	destFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create dest file: %w", err)
	}
	defer destFile.Close()

	// Copy contents
	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}
