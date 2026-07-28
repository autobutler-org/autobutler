package migration

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
)

// GoogleAPIClient uses real Google APIs to download data
type GoogleAPIClient struct {
	token       *oauth2.Token
	oauthConfig *oauth2.Config
	email       string
}

// NewGoogleAPIClient creates a client that uses real Google APIs
func NewGoogleAPIClient(token *oauth2.Token, oauthConfig *oauth2.Config, email string) *GoogleAPIClient {
	return &GoogleAPIClient{
		token:       token,
		oauthConfig: oauthConfig,
		email:       email,
	}
}

// RequestExport starts downloading data from Google APIs
func (c *GoogleAPIClient) RequestExport(ctx context.Context, services []string) (*ExportJob, error) {
	job := &ExportJob{
		ID:        fmt.Sprintf("api-export-%d", time.Now().Unix()),
		Status:    ExportStatusPending,
		Services:  services,
		CreatedAt: time.Now(),
	}

	return job, nil
}

// GetExportStatus returns immediate status (we download directly, no waiting)
func (c *GoogleAPIClient) GetExportStatus(ctx context.Context, exportID string) (*ExportJob, error) {
	now := time.Now()
	return &ExportJob{
		ID:          exportID,
		Status:      ExportStatusCompleted,
		CompletedAt: &now,
	}, nil
}

// DownloadArchive is not used by the API client
func (c *GoogleAPIClient) DownloadArchive(ctx context.Context, exportID string, archiveIndex int) (io.ReadCloser, error) {
	return nil, errors.New("not implemented for API client")
}

// ListArchives is not used by the API client
func (c *GoogleAPIClient) ListArchives(ctx context.Context, exportID string) ([]*ArchiveInfo, error) {
	return []*ArchiveInfo{}, nil
}

// DownloadToZip downloads all requested services and creates a ZIP file
func (c *GoogleAPIClient) DownloadToZip(ctx context.Context, services []string, zipWriter *zip.Writer, progressCallback func(service, fileName string, current, total int, currentBytes, totalBytes int64)) error {
	var totalBytes int64
	var downloadedBytes int64

	for _, service := range services {
		switch service {
		case "drive":
			if err := c.downloadDriveToZip(ctx, zipWriter, &downloadedBytes, &totalBytes, progressCallback); err != nil {
				return fmt.Errorf("failed to download Drive: %w", err)
			}
		case "photos":
			if err := c.downloadPhotosToZip(ctx, zipWriter, &downloadedBytes, &totalBytes, progressCallback); err != nil {
				return fmt.Errorf("failed to download Photos: %w", err)
			}
		default:
			log.Printf("Warning: service %s not yet implemented", service)
		}
	}

	return nil
}

func (c *GoogleAPIClient) downloadDriveToZip(ctx context.Context, zipWriter *zip.Writer, downloadedBytes, totalBytes *int64, progressCallback func(service, fileName string, current, total int, currentBytes, totalBytes int64)) error {
	log.Printf("Downloading Google Drive files for %s", c.email)

	driveClient, err := NewGoogleDriveClient(ctx, c.token, c.oauthConfig)
	if err != nil {
		return fmt.Errorf("failed to create Drive client: %w", err)
	}

	// First pass: download all files and calculate size
	files, err := driveClient.DownloadAllFiles(ctx, func(fileName string, current, total int) {
		// Report progress during file listing
		if progressCallback != nil {
			progressCallback("drive", fileName, current, total, *downloadedBytes, *totalBytes)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to download files: %w", err)
	}

	// Calculate total bytes from all files
	for _, file := range files {
		*totalBytes += file.Size
	}

	log.Printf("Downloaded %d files from Drive (%.2f MB total), adding to ZIP", len(files), float64(*totalBytes)/1024/1024)

	// Add files to ZIP and track bytes written
	for i, file := range files {
		// Create entry in ZIP under Drive folder
		zipPath := filepath.Join("Google Drive", file.Name)

		writer, err := zipWriter.Create(zipPath)
		if err != nil {
			file.Content.Close()
			return fmt.Errorf("failed to create ZIP entry for %s: %w", file.Name, err)
		}

		written, err := io.Copy(writer, file.Content)
		file.Content.Close()
		if err != nil {
			return fmt.Errorf("failed to write %s to ZIP: %w", file.Name, err)
		}

		*downloadedBytes += written

		// Report progress
		if progressCallback != nil {
			progressCallback("drive", file.Name, i+1, len(files), *downloadedBytes, *totalBytes)
		}

		log.Printf("Added to ZIP: %s (%.2f MB)", zipPath, float64(written)/1024/1024)
	}

	return nil
}

func (c *GoogleAPIClient) downloadPhotosToZip(ctx context.Context, zipWriter *zip.Writer, downloadedBytes, totalBytes *int64, progressCallback func(service, fileName string, current, total int, currentBytes, totalBytes int64)) error {
	log.Printf("Downloading Google Photos for %s", c.email)

	photosClient, err := NewGooglePhotosClient(ctx, c.token, c.oauthConfig)
	if err != nil {
		return fmt.Errorf("failed to create Photos client: %w", err)
	}

	photos, err := photosClient.DownloadAllPhotos(ctx, func(fileName string, current, total int) {
		if progressCallback != nil {
			progressCallback("photos", fileName, current, total, *downloadedBytes, *totalBytes)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to download photos: %w", err)
	}

	// Calculate total bytes (photos don't have size metadata, estimate from content)
	log.Printf("Downloaded %d photos, adding to ZIP", len(photos))

	// Add photos to ZIP and track bytes
	for i, photo := range photos {
		// Create entry in ZIP under Photos folder
		zipPath := filepath.Join("Google Photos", photo.Filename)

		writer, err := zipWriter.Create(zipPath)
		if err != nil {
			photo.Content.Close()
			return fmt.Errorf("failed to create ZIP entry for %s: %w", photo.Filename, err)
		}

		written, err := io.Copy(writer, photo.Content)
		photo.Content.Close()
		if err != nil {
			return fmt.Errorf("failed to write %s to ZIP: %w", photo.Filename, err)
		}

		*downloadedBytes += written
		*totalBytes += written // For photos, we don't know size until downloaded

		// Report progress
		if progressCallback != nil {
			progressCallback("photos", photo.Filename, i+1, len(photos), *downloadedBytes, *totalBytes)
		}

		log.Printf("Added to ZIP: %s (%.2f MB)", zipPath, float64(written)/1024/1024)
	}

	return nil
}
