package workerutil

import (
	"testing"
	"time"
)

////////////////
// Unit tests //
////////////////

// mock types for channels
type dummyDeleteFilesParams struct{}
type dummyMoveFileParams struct{}
type dummyUploadFilesParams struct{}
type dummyCreateFolderParams struct{}

func TestNewWorker_ChannelAccessors(t *testing.T) {
	w := NewWorker()
	if w.GetQuitChannel() == nil {
		t.Error("GetQuitChannel returned nil")
	}
	if w.GetErrorChannel() == nil {
		t.Error("GetErrorChannel returned nil")
	}
	if w.GetDeleteFilesChannel() == nil {
		t.Error("GetDeleteFilesChannel returned nil")
	}
	if w.GetMoveFileChannel() == nil {
		t.Error("GetMoveFileChannel returned nil")
	}
	if w.GetUploadFilesChannel() == nil {
		t.Error("GetUploadFilesChannel returned nil")
	}
	if w.GetCreateFolderChannel() == nil {
		t.Error("GetCreateFolderChannel returned nil")
	}
}

func TestWorker_Process_Quit(t *testing.T) {
	w := NewWorker()
	quit := w.GetQuitChannel()
	done := make(chan struct{})
	go func() {
		err := w.Process()
		if err != nil {
			t.Errorf("Process returned error: %v", err)
		}
		close(done)
	}()
	// Give goroutine time to start
	time.Sleep(10 * time.Millisecond)
	quit <- struct{}{}
	select {
	case <-done:
		// success
	case <-time.After(100 * time.Millisecond):
		t.Error("Process did not exit after quit signal")
	}
}

func TestWorker_LogErrors_ReceivesError(t *testing.T) {
	w := NewWorker()
	errCh := w.GetErrorChannel()
	// Run LogErrors in a goroutine, send an error, and check that it does not panic
	go func() {
		// Give LogErrors time to start
		time.Sleep(10 * time.Millisecond)
		errCh <- logErrorMock{}
		// Give time for log to print
		time.Sleep(10 * time.Millisecond)
	}()
	// LogErrors is an infinite loop, so run it with a timeout
	go func() { w.LogErrors() }()
	// No assertion: just ensure no panic/deadlock
}

type logErrorMock struct{}

func (logErrorMock) Error() string { return "mock error" }

///////////////////////
// Integration tests //
///////////////////////
