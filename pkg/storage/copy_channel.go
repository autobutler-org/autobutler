package storage

import (
	"errors"
)

// CopyTask represents a single file copy operation submitted to the worker pool.
type CopyTask struct {
	JobID  string
	Src    string
	Dest   string
	Result chan error
}

var copyQueue chan *CopyTask

func init() {
	// buffered queue and a small pool of workers
	copyQueue = make(chan *CopyTask, 1000)
	for i := 0; i < 2; i++ {
		go copyWorker()
	}
}

// SubmitCopyTask enqueues a copy task for processing. Returns error if queue is full.
func SubmitCopyTask(t *CopyTask) error {
	select {
	case copyQueue <- t:
		return nil
	default:
		return errors.New("copy queue full")
	}
}

func copyWorker() {
	for t := range copyQueue {
		// perform the actual file copy using package-level copyFile
		err := copyFile(t.Src, t.Dest)
		// deliver result and close channel
		if t.Result != nil {
			t.Result <- err
			close(t.Result)
		}
	}
}
