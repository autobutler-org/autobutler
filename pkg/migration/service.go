package migration

import (
	"errors"
	"context"
	"fmt"
	"time"
)

// Service implements GoogleTakeoutService
type Service struct {
	client    GoogleTakeoutClient
	store     ImportJobStore
	uploader  FileUploader
	extractor ArchiveExtractor
}

// NewService creates a new Google Takeout service
func NewService(
	client GoogleTakeoutClient,
	store ImportJobStore,
	uploader FileUploader,
	extractor ArchiveExtractor,
) *Service {
	return &Service{
		client:    client,
		store:     store,
		uploader:  uploader,
		extractor: extractor,
	}
}

// StartImport initiates a Google Takeout import
func (s *Service) StartImport(ctx context.Context, services []string) (*ImportJob, error) {
	if len(services) == 0 {
		return nil, errors.New("no services specified")
	}

	// Request export from Google Takeout client
	exportJob, err := s.client.RequestExport(ctx, services)
	if err != nil {
		return nil, fmt.Errorf("failed to request export: %w", err)
	}

	// Create import job in our system
	job := &ImportJob{
		ID:        fmt.Sprintf("import-%d", time.Now().Unix()),
		ExportID:  exportJob.ID,
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
func (s *Service) GetImportStatus(ctx context.Context, jobID string) (*ImportJob, error) {
	job, err := s.store.Get(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get import job: %w", err)
	}
	return job, nil
}

// ProcessImport handles the background processing of an import job
func (s *Service) ProcessImport(ctx context.Context, jobID string) error {
	job, err := s.store.Get(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get import job: %w", err)
	}

	// Update status to waiting for export
	job.Status = ImportStatusWaitingExport
	job.UpdatedAt = time.Now()
	if err := s.store.Update(ctx, job); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Wait for export completion (with timeout)
	exportJob, err := s.waitForExport(ctx, job.ExportID, 24*time.Hour)
	if err != nil {
		job.Status = ImportStatusFailed
		job.ErrorMsg = err.Error()
		job.UpdatedAt = time.Now()
		s.store.Update(ctx, job)
		return fmt.Errorf("export failed: %w", err)
	}

	if exportJob.Status == ExportStatusFailed {
		job.Status = ImportStatusFailed
		job.ErrorMsg = exportJob.ErrorMsg
		job.UpdatedAt = time.Now()
		s.store.Update(ctx, job)
		return fmt.Errorf("export failed: %s", exportJob.ErrorMsg)
	}

	// Download archives
	job.Status = ImportStatusDownloading
	job.UpdatedAt = time.Now()
	if err := s.store.Update(ctx, job); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	archives, err := s.client.ListArchives(ctx, job.ExportID)
	if err != nil {
		job.Status = ImportStatusFailed
		job.ErrorMsg = err.Error()
		job.UpdatedAt = time.Now()
		s.store.Update(ctx, job)
		return fmt.Errorf("failed to list archives: %w", err)
	}

	job.TotalFiles = len(archives)
	s.store.Update(ctx, job)

	// Process each archive
	for i, archive := range archives {
		if err := s.processArchive(ctx, job, archive); err != nil {
			job.Status = ImportStatusFailed
			job.ErrorMsg = err.Error()
			job.UpdatedAt = time.Now()
			s.store.Update(ctx, job)
			return fmt.Errorf("failed to process archive %d: %w", i, err)
		}

		job.FilesProcessed++
		job.Progress = float64(job.FilesProcessed) / float64(job.TotalFiles)
		job.UpdatedAt = time.Now()
		s.store.Update(ctx, job)
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

	return nil
}

// waitForExport polls the client until export is complete or timeout
func (s *Service) waitForExport(ctx context.Context, exportID string, timeout time.Duration) (*ExportJob, error) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	timeoutChan := time.After(timeout)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeoutChan:
			return nil, fmt.Errorf("export timed out after %v", timeout)
		case <-ticker.C:
			exportJob, err := s.client.GetExportStatus(ctx, exportID)
			if err != nil {
				return nil, fmt.Errorf("failed to get export status: %w", err)
			}

			if exportJob.Status == ExportStatusCompleted || exportJob.Status == ExportStatusFailed {
				return exportJob, nil
			}
		}
	}
}

// processArchive downloads, extracts, and uploads a single archive
func (s *Service) processArchive(ctx context.Context, job *ImportJob, archive *ArchiveInfo) error {
	// Download archive
	archiveReader, err := s.client.DownloadArchive(ctx, job.ExportID, archive.Index)
	if err != nil {
		return fmt.Errorf("failed to download archive: %w", err)
	}
	defer archiveReader.Close()

	// Extract archive to temporary directory
	job.Status = ImportStatusExtracting
	job.UpdatedAt = time.Now()
	s.store.Update(ctx, job)

	tempDir := fmt.Sprintf("/tmp/google-takeout-%s-%d", job.ID, archive.Index)
	if err := s.extractor.Extract(ctx, archiveReader, tempDir); err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	// Upload extracted files to cirrus
	job.Status = ImportStatusUploading
	job.UpdatedAt = time.Now()
	s.store.Update(ctx, job)

	if err := s.uploader.UploadDirectory(ctx, tempDir, "/Google Takeout", ""); err != nil {
		return fmt.Errorf("failed to upload files: %w", err)
	}

	return nil
}
