package migration

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
)

// CirrusFileUploader uploads files to cirrus storage
type CirrusFileUploader struct {
	deviceSerial string
}

// NewCirrusFileUploader creates a new cirrus file uploader
func NewCirrusFileUploader(deviceSerial string) *CirrusFileUploader {
	return &CirrusFileUploader{
		deviceSerial: deviceSerial,
	}
}

// UploadFile uploads a single file to cirrus storage
func (u *CirrusFileUploader) UploadFile(ctx context.Context, filePath string, content io.Reader, deviceSerial string) error {
	serial := deviceSerial
	if serial == "" {
		serial = u.deviceSerial
	}

	// Create a buffer to write our multipart data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Create form file
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	// Copy content to form
	if _, err := io.Copy(part, content); err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}

	writer.Close()

	// Use storageutil to upload
	err = storageutil.UploadFilesStreamed(storageutil.UploadFilesStreamedParams{
		Reader:       multipart.NewReader(&buf, writer.Boundary()),
		RootDir:      filepath.Dir(filePath),
		DeviceSerial: serial,
	})

	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	return nil
}

// UploadDirectory uploads an entire directory recursively
func (u *CirrusFileUploader) UploadDirectory(ctx context.Context, sourcePath string, destPath string, deviceSerial string) error {
	serial := deviceSerial
	if serial == "" {
		serial = u.deviceSerial
	}

	// Walk the directory tree
	return filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories (they'll be created automatically)
		if info.IsDir() {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Construct destination path
		fullDestPath := filepath.Join(destPath, relPath)

		// Open file
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", path, err)
		}
		defer file.Close()

		// Upload file
		if err := u.UploadFile(ctx, fullDestPath, file, serial); err != nil {
			return fmt.Errorf("failed to upload %s: %w", relPath, err)
		}

		return nil
	})
}
