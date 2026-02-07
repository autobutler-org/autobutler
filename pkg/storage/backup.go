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
	Files        []string     `json:"files,omitempty"`

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

	// build per-source device file lists
	filesByDevice := make(map[string][]string)
	cirrusByDevice := make(map[string]string)
	total := 0
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
		if d.CirrusDir == "" {
			continue
		}
		cirrusByDevice[serial] = d.CirrusDir
		files := make([]string, 0)
		_ = filepath.Walk(d.CirrusDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			files = append(files, path)
			return nil
		})
		if len(files) == 0 {
			continue
		}
		filesByDevice[serial] = files
		total += len(files)
	}

	mu.Lock()
	job.TotalFiles = total
	// store list of gathered files for optional inclusion in status
	allFiles := make([]string, 0, total)
	for _, fs := range filesByDevice {
		allFiles = append(allFiles, fs...)
	}
	job.Files = allFiles
	mu.Unlock()

	// submit backup tasks (one per source device) to the backup worker channel
	results := make([]chan error, 0, len(filesByDevice))
	counts := make([]int, 0, len(filesByDevice))
	for serial, files := range filesByDevice {
		select {
		case <-ctx.Done():
			finishJobCanceled(job)
			return
		default:
		}
		srcBase := cirrusByDevice[serial]
		task := &BackupTask{JobID: job.ID, SrcDir: srcBase, DestDir: target.CirrusDir, Result: make(chan error, 1)}
		if err := SubmitBackupTask(task); err != nil {
			// queue full or submit failed; count all files for this device as processed but continue
			mu.Lock()
			job.FilesCopied += len(files)
			mu.Unlock()
			continue
		}
		results = append(results, task.Result)
		counts = append(counts, len(files))
	}

	// wait for all results (or cancellation)
	for i, res := range results {
		select {
		case <-ctx.Done():
			finishJobCanceled(job)
			return
		case err := <-res:
			_ = err // currently just count successes/failures uniformly; could log
			mu.Lock()
			job.FilesCopied += counts[i]
			mu.Unlock()
		}
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
