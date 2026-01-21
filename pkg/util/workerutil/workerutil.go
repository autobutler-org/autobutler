package workerutil

import (
	"autobutler/pkg/util/storageutil"
	"log"
)

type Worker interface {
	Process() error
	GetQuitChannel() chan struct{}
	GetErrorChannel() chan error
	LogErrors() error
	GetDeleteFilesChannel() storageutil.DeleteFilesChannel
	GetMoveFileChannel() storageutil.MoveFileChannel
	GetUploadFilesChannel() storageutil.UploadFilesChannel
	GetCreateFolderChannel() storageutil.CreateFolderChannel
}

type worker struct {
	quitChannel         chan struct{}
	errorChannel        chan error
	deleteFilesChannel  storageutil.DeleteFilesChannel
	moveFileChannel     storageutil.MoveFileChannel
	uploadFilesChannel  storageutil.UploadFilesChannel
	createFolderChannel storageutil.CreateFolderChannel
}

func NewWorker() Worker {
	return &worker{
		quitChannel:         make(chan struct{}),
		errorChannel:        make(chan error),
		deleteFilesChannel:  make(storageutil.DeleteFilesChannel),
		moveFileChannel:     make(storageutil.MoveFileChannel),
		uploadFilesChannel:  make(storageutil.UploadFilesChannel),
		createFolderChannel: make(storageutil.CreateFolderChannel),
	}
}

func (w *worker) Process() error {
	for {
		select {
		case deleteReq := <-w.deleteFilesChannel:
			if _, err := storageutil.DeleteFiles(deleteReq); err != nil {
				w.errorChannel <- err
			}
		case moveReq := <-w.moveFileChannel:
			if _, err := storageutil.MoveFile(moveReq); err != nil {
				w.errorChannel <- err
			}
		case uploadReq := <-w.uploadFilesChannel:
			if _, err := storageutil.UploadFiles(uploadReq); err != nil {
				w.errorChannel <- err
			}
		case createFolderReq := <-w.createFolderChannel:
			if _, err := storageutil.CreateFolder(createFolderReq); err != nil {
				w.errorChannel <- err
			}
		case <-w.quitChannel:
			return nil
		}
	}
}

func (w *worker) GetQuitChannel() chan struct{} {
	return w.quitChannel
}

func (w *worker) GetErrorChannel() chan error {
	return w.errorChannel
}

func (w *worker) LogErrors() error {
	for {
		err := <-w.errorChannel
		if err != nil {
			log.Printf("Worker service error: %v\n", err)
		}
	}
}

func (w *worker) GetDeleteFilesChannel() storageutil.DeleteFilesChannel {
	return w.deleteFilesChannel
}

func (w *worker) GetMoveFileChannel() storageutil.MoveFileChannel {
	return w.moveFileChannel
}

func (w *worker) GetUploadFilesChannel() storageutil.UploadFilesChannel {
	return w.uploadFilesChannel
}

func (w *worker) GetCreateFolderChannel() storageutil.CreateFolderChannel {
	return w.createFolderChannel
}
