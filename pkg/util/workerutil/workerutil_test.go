package workerutil

import (
	"autobutler/pkg/util/cirrusutil"
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func startWorkerAndQuitOnDone(t *testing.T, fn func(w Worker)) {
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
	fn(w)
	quit <- struct{}{}
	select {
	case <-done:
		// success
	case <-time.After(200 * time.Millisecond):
		t.Error("Worker did not exit after quit signal")
	}
}

func TestWorkerIntegration_DeleteFile(t *testing.T) {
	testFileName := "integration_deletefile.txt"
	cirrusDir := cirrusutil.GetCirrusDir()

	filePath := filepath.Join(cirrusDir, testFileName)
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	startWorkerAndQuitOnDone(t, func(w Worker) {
		deleteCh := w.GetDeleteFilesChannel()
		deleteCh <- cirrusutil.DeleteFilesParams{
			RootDir:   "",
			FilePaths: []string{testFileName},
		}
	})

	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("file was not deleted: %v", err)
	}
}

func TestWorkerIntegration_MoveFile(t *testing.T) {
	cirrusDir := cirrusutil.GetCirrusDir()
	oldName := "integration_movefile_old.txt"
	newName := "integration_movefile_new.txt"
	oldPath := filepath.Join(cirrusDir, oldName)
	newPath := filepath.Join(cirrusDir, newName)

	if err := os.WriteFile(oldPath, []byte("move me"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	startWorkerAndQuitOnDone(t, func(w Worker) {
		moveCh := w.GetMoveFileChannel()
		moveCh <- cirrusutil.MoveFileParams{
			FilePath:    oldName,
			NewFilePath: newName,
		}
	})

	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("file was not moved: %v", err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("old file still exists: %v", err)
	}

	// Cleanup
	os.Remove(oldPath)
	os.Remove(newPath)
}

func TestWorkerIntegration_CreateFolder(t *testing.T) {
	cirrusDir := cirrusutil.GetCirrusDir()
	folderName := "integration_created_folder"
	folderPath := filepath.Join(cirrusDir, folderName)

	startWorkerAndQuitOnDone(t, func(w Worker) {
		createCh := w.GetCreateFolderChannel()
		createCh <- cirrusutil.CreateFolderParams{
			FolderDir:  "",
			FolderName: folderName,
		}
	})

	if _, err := os.Stat(folderPath); err != nil {
		t.Errorf("folder was not created: %v", err)
	}

	// Cleanup
	os.RemoveAll(folderPath)
}

func TestWorkerIntegration_UploadFile(t *testing.T) {
	cirrusDir := cirrusutil.GetCirrusDir()
	fileName := "integration_uploadfile.txt"
	fileContent := []byte("upload test content")

	// Create a multipart form and parse it to get a real *multipart.FileHeader
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	fw, err := mw.CreateFormFile("file", fileName)
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := fw.Write(fileContent); err != nil {
		t.Fatalf("failed to write file content: %v", err)
	}
	mw.Close()

	req := httptest.NewRequest("POST", "/", &b)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if err := req.ParseMultipartForm(10 << 20); err != nil {
		t.Fatalf("failed to parse multipart form: %v", err)
	}
	fh := req.MultipartForm.File["file"][0]

	startWorkerAndQuitOnDone(t, func(w Worker) {
		uploadCh := w.GetUploadFilesChannel()
		uploadCh <- cirrusutil.UploadFilesParams{
			RootDir:     "",
			FileHeaders: []*multipart.FileHeader{fh},
		}
	})

	uploadedPath := filepath.Join(cirrusDir, fileName)
	data, err := os.ReadFile(uploadedPath)
	if err != nil {
		t.Errorf("uploaded file not found: %v", err)
	} else if !bytes.Equal(data, fileContent) {
		t.Errorf("uploaded file content mismatch: got %q, want %q", data, fileContent)
	}
	os.Remove(uploadedPath)
}
