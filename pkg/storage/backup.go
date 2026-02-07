package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"autobutler/pkg/util/storageutil"
	"github.com/google/uuid"
)

type BackupStatus string

const (
	BackupPending    BackupStatus = "PENDING"
	BackupProcessing BackupStatus = "PROCESSING"
	BackupCompleted  BackupStatus = "COMPLETED"
	BackupFailed     BackupStatus = "FAILED"
	BackupCanceled   BackupStatus = "CANCELED"
)

type BackupJob struct {
	ID           string       `json:"id"`
	TargetSerial string       `json:"targetSerial"`
	Status       BackupStatus `json:"status"`
	CreatedAt    time.Time    `json:"createdAt"`
	StartedAt    *time.Time   `json:"startedAt,omitempty"`
	CompletedAt  *time.Time   `json:"completedAt,omitempty"`
	ErrorMsg     string       `json:"errorMsg,omitempty"`
	TotalFiles   int          `json:"totalFiles,omitempty"`
	FilesCopied  int          `json:"filesCopied,omitempty"`

	cancel context.CancelFunc `json:"-"`
}

// in-memory store
var (
	mu sync.RWMutex
	// jobs by id
	jobs = make(map[string]*BackupJob)
	// map target serial to list of job ids
	jobsByTarget = make(map[string][]string)
)

func CreateBackupJob(targetSerial string) (string, error) {
	// ensure target exists
	devices, err := storageutil.GetManagedDevices()
	if err != nil {
		return "", err
	}
	var target *storageutil.ManagedDevice
	for _, d := range devices {
		if d.UsbInfo != nil && d.UsbInfo.GetSerial() == targetSerial {
			target = &d
			break
		}
	}
	if target == nil {
		return "", errors.New("target device not found")
	}

	id := uuid.New().String()
	job := &BackupJob{
		ID:           id,
		TargetSerial: targetSerial,
		Status:       BackupPending,
		CreatedAt:    time.Now(),
	}
	mu.Lock()
	jobs[id] = job
	jobsByTarget[targetSerial] = append(jobsByTarget[targetSerial], id)
	mu.Unlock()

	// start job in background
	ctx, cancel := context.WithCancel(context.Background())
	job.cancel = cancel
	go runBackupJob(ctx, job)
	return id, nil
}

func runBackupJob(ctx context.Context, job *BackupJob) {
	start := time.Now()
	now := start
	mu.Lock()
	job.Status = BackupProcessing
	job.StartedAt = &now
	mu.Unlock()

	// gather devices and files to copy
	devices, err := storageutil.GetManagedDevices()
	if err != nil {
		finishJobWithError(job, fmt.Errorf("failed to list devices: %w", err))
		return
	}
	prefs, _ := storageutil.LoadDevicePrefs()

	// locate target device
	var target *storageutil.ManagedDevice
	for _, d := range devices {
		if d.UsbInfo != nil && d.UsbInfo.GetSerial() == job.TargetSerial {
			target = &d
			break
		}
	}
	if target == nil {
		finishJobWithError(job, errors.New("target device disappeared"))
		return
	}

	// build list of source files
	sources := make([]string, 0)
	for _, d := range devices {
		if d.UsbInfo == nil {
			continue
		}
		serial := d.UsbInfo.GetSerial()
		if serial == job.TargetSerial {
			continue
		}
		if prefs.IsBackup(serial) {
			continue
		}
		// walk cirrus dir
		if d.CirrusDir == "" {
			continue
		}
		_ = filepath.Walk(d.CirrusDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			sources = append(sources, path)
			return nil
		})
	}

	mu.Lock()
	job.TotalFiles = len(sources)
	mu.Unlock()

	// perform copies
	for _, src := range sources {
		select {
		case <-ctx.Done():
			finishJobCanceled(job)
			return
		default:
		}
		// build relative path from source cirrus base
		var base string
		var rel string
		for _, d := range devices {
			if d.CirrusDir != "" && filepath.HasPrefix(src, filepath.Clean(d.CirrusDir)) {
				base = d.CirrusDir
				break
			}
		}
		if base == "" {
			// can't determine relative path; skip
			mu.Lock()
			job.FilesCopied++
			mu.Unlock()
			continue
		}
		rel, _ = filepath.Rel(base, src)
		dest := filepath.Join(target.CirrusDir, rel)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			finishJobWithError(job, fmt.Errorf("mkdir failed: %w", err))
			return
		}
		if err := copyFile(src, dest); err != nil {
			// log and continue
			_ = err
		}
		mu.Lock()
		job.FilesCopied++
		mu.Unlock()
	}

	finishJobCompleted(job)
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	// use temp file then rename
	tmp, err := ioutil.TempFile(filepath.Dir(dest), ".tmp-copy-")
	if err != nil {
		return err
	}
	defer func() { tmp.Close(); os.Remove(tmp.Name()) }()
	if _, err := io.Copy(tmp, in); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), dest); err != nil {
		return err
	}
	return nil
}

func finishJobWithError(job *BackupJob, err error) {
	mu.Lock()
	defer mu.Unlock()
	job.Status = BackupFailed
	job.ErrorMsg = err.Error()
	t := time.Now()
	job.CompletedAt = &t
}

func finishJobCompleted(job *BackupJob) {
	mu.Lock()
	defer mu.Unlock()
	job.Status = BackupCompleted
	t := time.Now()
	job.CompletedAt = &t
}

func finishJobCanceled(job *BackupJob) {
	mu.Lock()
	defer mu.Unlock()
	job.Status = BackupCanceled
	t := time.Now()
	job.CompletedAt = &t
}

func GetJobsForTarget(targetSerial string) ([]*BackupJob, error) {
	mu.RLock()
	defer mu.RUnlock()
	ids := jobsByTarget[targetSerial]
	out := make([]*BackupJob, 0, len(ids))
	for _, id := range ids {
		if j, ok := jobs[id]; ok {
			out = append(out, j)
		}
	}
	return out, nil
}

func CancelJobsForTarget(targetSerial string) error {
	mu.RLock()
	ids := append([]string{}, jobsByTarget[targetSerial]...)
	mu.RUnlock()
	for _, id := range ids {
		mu.RLock()
		j, ok := jobs[id]
		mu.RUnlock()
		if !ok {
			continue
		}
		if j.cancel != nil {
			j.cancel()
		}
	}
	return nil
}

func GetJobByID(id string) (*BackupJob, error) {
	mu.RLock()
	defer mu.RUnlock()
	j, ok := jobs[id]
	if !ok {
		return nil, fmt.Errorf("job %s not found", id)
	}
	return j, nil
}
