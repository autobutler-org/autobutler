package migration

import (
	"errors"
	"archive/zip"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
)

// GoogleAPIService implements direct API downloading
type GoogleAPIService struct {
	store       ImportJobStore
	cirrusPath  string
	oauthConfig *oauth2.Config
}

// NewGoogleAPIService creates a service that downloads directly from Google APIs
func NewGoogleAPIService(
	store ImportJobStore,
	cirrusPath string,
	oauthConfig *oauth2.Config,
) *GoogleAPIService {
	return &GoogleAPIService{
		store:       store,
		cirrusPath:  cirrusPath,
		oauthConfig: oauthConfig,
	}
}

// StartImport initiates a Google data download
func (s *GoogleAPIService) StartImport(ctx context.Context, services []string, email string, token *oauth2.Token) (*ImportJob, error) {
	if len(services) == 0 {
		return nil, errors.New("no services specified")
	}

	if token == nil {
		return nil, errors.New("no OAuth token provided")
	}

	// Create import job
	job := &ImportJob{
		ID:        fmt.Sprintf("import-%d", time.Now().Unix()),
		ExportID:  fmt.Sprintf("export-%d", time.Now().Unix()),
		Status:    ImportStatusInitiated,
		Services:  services,
		Progress:  0.0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.store.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create import job: %w", err)
	}

	return job, nil
}

// GetImportStatus returns the status of an import job
func (s *GoogleAPIService) GetImportStatus(ctx context.Context, jobID string) (*ImportJob, error) {
	job, err := s.store.Get(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get import job: %w", err)
	}
	return job, nil
}

// ProcessImport downloads data from Google APIs and creates a ZIP file
func (s *GoogleAPIService) ProcessImport(ctx context.Context, jobID string, email string, token *oauth2.Token) error {
	job, err := s.store.Get(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get import job: %w", err)
	}

	log.Printf("Starting Google API download for job %s, email %s", jobID, email)

	// Update status to downloading
	job.Status = ImportStatusDownloading
	job.UpdatedAt = time.Now()
	if err := s.store.Update(ctx, job); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Create ZIP file at Cirrus root
	zipFileName := fmt.Sprintf("Google_Data_%s_%d.zip", email, time.Now().Unix())
	zipPath := filepath.Join(s.cirrusPath, zipFileName)

	// Ensure cirrus directory exists
	if err := os.MkdirAll(s.cirrusPath, 0755); err != nil {
		job.Status = ImportStatusFailed
		job.ErrorMsg = fmt.Sprintf("failed to create cirrus directory: %v", err)
		job.UpdatedAt = time.Now()
		s.store.Update(ctx, job)
		return fmt.Errorf("failed to create cirrus directory: %w", err)
	}

	zipFile, err := os.Create(zipPath)
	if err != nil {
		job.Status = ImportStatusFailed
		job.ErrorMsg = fmt.Sprintf("failed to create ZIP file: %v", err)
		job.UpdatedAt = time.Now()
		s.store.Update(ctx, job)
		return fmt.Errorf("failed to create ZIP file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	log.Printf("Created ZIP file at: %s", zipPath)

	// Create API client
	apiClient := NewGoogleAPIClient(token, s.oauthConfig, email)

	// Download data and add to ZIP
	totalFiles := 0
	processedFiles := 0
	var downloadedBytes int64
	var totalBytes int64

	progressCallback := func(service, fileName string, current, total int, currentBytes, totalSizeBytes int64) {
		log.Printf("[%s] Downloading %s (%d/%d, %.2f MB / %.2f MB)",
			service, fileName, current, total,
			float64(currentBytes)/1024/1024, float64(totalSizeBytes)/1024/1024)

		processedFiles = current
		totalFiles = total
		downloadedBytes = currentBytes
		totalBytes = totalSizeBytes

		if totalFiles > 0 {
			job.FilesProcessed = processedFiles
			job.TotalFiles = totalFiles
			job.DownloadedBytes = downloadedBytes
			job.TotalBytes = totalBytes
			job.Progress = float64(processedFiles) / float64(totalFiles)
			job.UpdatedAt = time.Now()
			s.store.Update(ctx, job)
		}
	}

	if err := apiClient.DownloadToZip(ctx, job.Services, zipWriter, progressCallback); err != nil {
		job.Status = ImportStatusFailed
		job.ErrorMsg = fmt.Sprintf("download failed: %v", err)
		job.UpdatedAt = time.Now()
		s.store.Update(ctx, job)
		return fmt.Errorf("download failed: %w", err)
	}

	// Close ZIP writer to finalize
	if err := zipWriter.Close(); err != nil {
		job.Status = ImportStatusFailed
		job.ErrorMsg = fmt.Sprintf("failed to finalize ZIP: %v", err)
		job.UpdatedAt = time.Now()
		s.store.Update(ctx, job)
		return fmt.Errorf("failed to finalize ZIP: %w", err)
	}

	// Mark as completed
	now := time.Now()
	job.Status = ImportStatusCompleted
	job.CompletedAt = &now
	job.Progress = 1.0
	job.UpdatedAt = now
	if err := s.store.Update(ctx, job); err != nil {
		return fmt.Errorf("failed to mark job as completed: %w", err)
	}

	log.Printf("Import job %s completed successfully. ZIP saved to: %s", jobID, zipPath)

	return nil
}
