package migration

import (
	"context"
	"io"
	"time"
)

type ExportJobStatus string

const (
	ExportStatusPending    ExportJobStatus = "PENDING"
	ExportStatusProcessing ExportJobStatus = "PROCESSING"
	ExportStatusCompleted  ExportJobStatus = "COMPLETED"
	ExportStatusFailed     ExportJobStatus = "FAILED"
)

type ExportJob struct {
	ID          string
	Status      ExportJobStatus
	Services    []string
	CreatedAt   time.Time
	CompletedAt *time.Time
	ErrorMsg    string
}

type ArchiveInfo struct {
	Index       int
	Size        int64
	DownloadURL string
}

type ImportJobStatus string

const (
	ImportStatusInitiated     ImportJobStatus = "INITIATED"
	ImportStatusWaitingExport ImportJobStatus = "WAITING_FOR_EXPORT"
	ImportStatusDownloading   ImportJobStatus = "DOWNLOADING"
	ImportStatusExtracting    ImportJobStatus = "EXTRACTING"
	ImportStatusUploading     ImportJobStatus = "UPLOADING"
	ImportStatusCompleted     ImportJobStatus = "COMPLETED"
	ImportStatusFailed        ImportJobStatus = "FAILED"
)

type ImportJob struct {
	ID             string
	ExportID       string
	Status         ImportJobStatus
	Services       []string
	Progress       float64
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CompletedAt    *time.Time
	ErrorMsg       string
	FilesProcessed int
	TotalFiles     int
}

type GoogleTakeoutClient interface {
	RequestExport(ctx context.Context, services []string) (*ExportJob, error)
	GetExportStatus(ctx context.Context, exportID string) (*ExportJob, error)
	DownloadArchive(ctx context.Context, exportID string, archiveIndex int) (io.ReadCloser, error)
	ListArchives(ctx context.Context, exportID string) ([]*ArchiveInfo, error)
}

type GoogleTakeoutService interface {
	StartImport(ctx context.Context, services []string) (*ImportJob, error)
	GetImportStatus(ctx context.Context, jobID string) (*ImportJob, error)
	ProcessImport(ctx context.Context, jobID string) error
}

type ImportJobStore interface {
	Create(ctx context.Context, job *ImportJob) error
	Get(ctx context.Context, jobID string) (*ImportJob, error)
	Update(ctx context.Context, job *ImportJob) error
	List(ctx context.Context) ([]*ImportJob, error)
}

type FileUploader interface {
	UploadFile(ctx context.Context, filePath string, content io.Reader, deviceSerial string) error
	UploadDirectory(ctx context.Context, sourcePath string, destPath string, deviceSerial string) error
}

type ArchiveExtractor interface {
	Extract(ctx context.Context, archive io.Reader, destDir string) error
	ListContents(ctx context.Context, archive io.Reader) ([]string, error)
}
