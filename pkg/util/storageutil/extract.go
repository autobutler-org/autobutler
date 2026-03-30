package storageutil

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	// MaxZipEntryBytes is the maximum number of bytes that will be written per
	// zip entry. This guards against zip bombs without rejecting legitimate large
	// files such as multi-GB Takeout videos.
	MaxZipEntryBytes int64 = 10 * 1024 * 1024 * 1024 // 10 GiB

	// MaxZipEntries is the maximum number of entries that will be extracted from
	// a single archive.
	MaxZipEntries = 100_000
)

// ExtractFileParams contains parameters for extracting an archive file
type ExtractFileParams struct {
	FilePath     string
	DeviceSerial string
}

// ExtractFileResult contains the result of an extraction operation
type ExtractFileResult struct {
	DestDir string
}

// ExtractFile extracts a zip archive in place on disk.
// Files are streamed one at a time; the zip central directory is the only
// data read up-front (as required by archive/zip.OpenReader).
func (s *StorageService) ExtractFile(params ExtractFileParams) (*ExtractFileResult, error) {
	device, err := s.FindManagedDeviceBySerial(params.DeviceSerial)
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}
	defaultCirrusDir, err := GetCirrusDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cirrus directory: %w", err)
	}
	return ExtractFileImpl(params, device, defaultCirrusDir)
}

// ExtractFileImpl extracts a zip archive using pre-resolved device and cirrus directory.
// Use this in tests to inject test devices without hitting the real filesystem detector.
func ExtractFileImpl(params ExtractFileParams, device *ManagedDevice, defaultCirrusDir string) (*ExtractFileResult, error) {
	cirrusDir := defaultCirrusDir
	if device != nil {
		cirrusDir = device.CirrusDir
	}

	fullPath := filepath.Join(cirrusDir, params.FilePath)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", params.FilePath)
	}

	fileType := DetermineFileTypeFromPath(fullPath)
	if fileType != FileTypeArchive {
		return nil, fmt.Errorf("file is not an archive: %s", params.FilePath)
	}

	ext := strings.ToLower(filepath.Ext(fullPath))
	if ext != ".zip" {
		return nil, fmt.Errorf("only zip archives are supported for extraction: %s", params.FilePath)
	}

	destDir := filepath.Join(filepath.Dir(fullPath), strings.TrimSuffix(filepath.Base(fullPath), filepath.Ext(fullPath)))
	destDir = GetNonConflictingPath(destDir)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err) // coverage: ignore - requires filesystem permission errors
	}

	r, err := zip.OpenReader(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file: %w", err)
	}
	defer r.Close()

	return extractZip(&r.Reader, destDir, MaxZipEntries, MaxZipEntryBytes)
}

func extractZip(r *zip.Reader, destDir string, maxEntries int, maxBytesPerEntry int64) (*ExtractFileResult, error) {
	if len(r.File) > maxEntries {
		return nil, fmt.Errorf("zip archive contains %d entries, which exceeds the limit of %d", len(r.File), maxEntries)
	}

	for _, f := range r.File {
		if err := extractZipEntry(f, destDir, maxBytesPerEntry); err != nil {
			return nil, err
		}
	}

	return &ExtractFileResult{
		DestDir: destDir,
	}, nil
}

func extractZipEntry(f *zip.File, destDir string, maxBytes int64) error {
	name := f.Name

	// Sanitize: reject entries that try to escape the destination directory.
	// Clean the name and ensure it doesn't start with "/" or contain "..".
	cleaned := filepath.Clean(name)
	if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
		return fmt.Errorf("invalid zip entry path: %s", name)
	}

	targetPath := filepath.Join(destDir, cleaned)

	// Ensure the resolved path is still within destDir (second guard).
	absDestDir, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("failed to resolve destination directory: %w", err) // coverage: ignore - requires filepath.Abs to fail
	}
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("failed to resolve target path: %w", err) // coverage: ignore - requires filepath.Abs to fail
	}
	if !strings.HasPrefix(absTarget, absDestDir+string(os.PathSeparator)) && absTarget != absDestDir {
		return fmt.Errorf("zip entry %q escapes destination directory", name)
	}

	if f.FileInfo().IsDir() {
		if err := os.MkdirAll(targetPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", targetPath, err) // coverage: ignore - requires filesystem permission errors
		}
		return nil
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %w", targetPath, err) // coverage: ignore - requires filesystem permission errors
	}

	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("failed to open zip entry %s: %w", name, err)
	}
	defer rc.Close()

	out, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", targetPath, err) // coverage: ignore - requires filesystem permission errors
	}
	defer out.Close()

	limited := io.LimitReader(rc, maxBytes+1)
	n, err := io.Copy(out, limited)
	if err != nil {
		return fmt.Errorf("failed to extract file %s: %w", name, err)
	}
	if n > maxBytes {
		return fmt.Errorf("zip entry %q exceeds maximum allowed size of %d bytes", name, maxBytes)
	}

	return nil
}
