package backup

import (
	"context"
	"time"
)

type BackupJobStatus string

const (
	BackupStatusPending   BackupJobStatus = "PENDING"
	BackupStatusScanning  BackupJobStatus = "SCANNING"
	BackupStatusCopying   BackupJobStatus = "COPYING"
	BackupStatusCompleted BackupJobStatus = "COMPLETED"
	BackupStatusFailed    BackupJobStatus = "FAILED"
)

type SourceDevice struct {
	Name      string
	Serial    string
	CirrusDir string
}

type SourceDeviceProgress struct {
	DeviceSerial string `json:"deviceSerial"`
	DeviceName   string `json:"deviceName"`
	FilesTotal   int    `json:"filesTotal"`
	FilesCopied  int    `json:"filesCopied"`
	FilesSkipped int    `json:"filesSkipped"`
	BytesTotal   int64  `json:"bytesTotal"`
	BytesCopied  int64  `json:"bytesCopied"`
}

type BackupJob struct {
	ID                 string                 `json:"id"`
	Status             BackupJobStatus        `json:"status"`
	TargetDeviceSerial string                 `json:"targetDeviceSerial"`
	Progress           float64                `json:"progress"`
	TotalFiles         int                    `json:"totalFiles"`
	FilesCopied        int                    `json:"filesCopied"`
	FilesSkipped       int                    `json:"filesSkipped"`
	TotalBytes         int64                  `json:"totalBytes"`
	BytesCopied        int64                  `json:"bytesCopied"`
	SourceDevices      []SourceDeviceProgress `json:"sourceDevices"`
	ErrorMsg           string                 `json:"errorMsg,omitempty"`
	CreatedAt          time.Time              `json:"createdAt"`
	UpdatedAt          time.Time              `json:"updatedAt"`
	CompletedAt        *time.Time             `json:"completedAt,omitempty"`
}

type BackupProgressData struct {
	JobID       string  `json:"jobId"`
	Progress    float64 `json:"progress"`
	FilesCopied int     `json:"filesCopied"`
	TotalFiles  int     `json:"totalFiles"`
	BytesCopied int64   `json:"bytesCopied"`
	TotalBytes  int64   `json:"totalBytes"`
	CurrentFile string  `json:"currentFile,omitempty"`
}

type BackupJobStore interface {
	Create(ctx context.Context, job *BackupJob) error
	Get(ctx context.Context, jobID string) (*BackupJob, error)
	Update(ctx context.Context, job *BackupJob) error
	List(ctx context.Context) ([]*BackupJob, error)
}
