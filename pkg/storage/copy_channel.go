package storage

import (
	"errors"
	"os"
	"path/filepath"
)

// BackupTask represents a backup operation from a source device directory to a target device directory.
type BackupTask struct {
	JobID   string
	SrcDir  string
	DestDir string
	Result  chan error
}

var backupQueue chan *BackupTask

func init() {
	// buffered queue and a small pool of workers
	backupQueue = make(chan *BackupTask, 100)
	for i := 0; i < 2; i++ {
		go backupWorker()
	}
}

// SubmitBackupTask enqueues a backup task for processing. Returns error if queue is full.
func SubmitBackupTask(t *BackupTask) error {
	select {
	case backupQueue <- t:
		return nil
	default:
		return errors.New("backup queue full")
	}
}

func backupWorker() {
	for t := range backupQueue {
		// perform the backup for the device directory
		err := runBackupForDevice(t.SrcDir, t.DestDir)
		// deliver result and close channel
		if t.Result != nil {
			t.Result <- err
			close(t.Result)
		}
	}
}

func runBackupForDevice(srcDir, destDir string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(srcDir, path)
		if relErr != nil {
			return nil
		}
		dest := filepath.Join(destDir, rel)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		// copyFile is available in this package
		if err := copyFile(path, dest); err != nil {
			return err
		}
		return nil
	})
}
