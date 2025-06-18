package update

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"
)

type UpdateRequest struct {
	Version string `json:"version"`
}

const binaryName = "autobutler"
var backupName = fmt.Sprintf("%s_backup", binaryName)
var extractedName = fmt.Sprintf("%s_extracted", binaryName)

func backupSelf() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	tmpFile, err := os.CreateTemp("", backupName)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	src, err := os.Open(execPath)
	if err != nil {
		return "", fmt.Errorf("failed to open current binary: %w", err)
	}
	defer src.Close()

	if _, err := src.Seek(0, 0); err != nil {
		return "", fmt.Errorf("failed to seek in current binary: %w", err)
	}
	if _, err := tmpFile.ReadFrom(src); err != nil {
		return "", fmt.Errorf("failed to copy binary to temp: %w", err)
	}
	return tmpFile.Name(), nil
}

func replaceSelf(body io.Reader) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	tmpFile, err := os.CreateTemp("", "autobutler_update_*")
	if err != nil {
		return fmt.Errorf("failed to create temp file for update: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.ReadFrom(body); err != nil {
		return fmt.Errorf("failed to write update to temp file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}
	// Rewind the temp file to the beginning
	if _, err := tmpFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek in temp file: %w", err)
	}

	gzReader, err := gzip.NewReader(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	var binFile *os.File
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}
		if header.Typeflag == tar.TypeReg && (header.Name == binaryName || header.Name == fmt.Sprintf("./%s", binaryName)) {
			binFile, err = os.CreateTemp("", extractedName)
			if err != nil {
				return fmt.Errorf("failed to create temp file for extracted binary: %w", err)
			}
			if _, err := io.Copy(binFile, tarReader); err != nil {
				return fmt.Errorf("failed to extract binary from tar: %w", err)
			}
			if err := binFile.Sync(); err != nil {
				return fmt.Errorf("failed to sync extracted binary: %w", err)
			}
			break
		}
	}
	if binFile == nil {
		return fmt.Errorf("binary not found in archive")
	}
	defer os.Remove(binFile.Name())
	defer binFile.Close()
	// Rewind the extracted binary file for reading
	if _, err := binFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek in extracted binary: %w", err)
	}

	// Overwrite the current executable with the new binary
	src, err := os.Open(binFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open temp update file: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(execPath, os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to open executable for writing: %w", err)
	}
	defer dst.Close()

	if _, err := src.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek in update file: %w", err)
	}
	if _, err := dst.ReadFrom(src); err != nil {
		return fmt.Errorf("failed to overwrite executable: %w", err)
	}
	return nil
}

func Update(version string) error {
    // Copy the current binary to some temporary system location
    // Download the new release
    // Unpack the new release, replacing the currently running binary location
	if version == "" {
		return nil // No update needed
	}
	_, err := backupSelf()
	if err != nil {
		return fmt.Errorf("failed to copy current binary: %w", err)
	}
	baseUrl := os.Getenv("AUTOBUTLER_UPDATE_URL")
	if baseUrl == "" {
		baseUrl = "https://github.com/exokomodo/autobutler.ai/releases/download"
	}
	url := fmt.Sprintf("%s/%s/autobutler_%s_%s.tar.gz", baseUrl, version, runtime.GOOS, runtime.GOARCH)
	fmt.Println("Downloading update from", url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to download update: %s", resp.Status)
	}
	if err := replaceSelf(resp.Body); err != nil {
		return fmt.Errorf("failed to replace self with update: %w", err)
	}
	fmt.Println("Update successful.")
	return nil
}

func RestartAutobutler(delay time.Duration) {
	time.Sleep(delay)
	os.Exit(0)
}
