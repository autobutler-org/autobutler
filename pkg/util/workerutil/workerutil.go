package workerutil

import (
	"autobutler/pkg/util/cirrusutil"
	"log"
)

type Worker interface {
	Process() error
	GetQuitChannel() chan struct{}
	GetErrorChannel() chan error
	LogErrors() error
	GetDeleteFilesChannel() cirrusutil.DeleteFilesChannel
	GetMoveFileChannel() cirrusutil.MoveFileChannel
	GetUploadFilesChannel() cirrusutil.UploadFilesChannel
	GetCreateFolderChannel() cirrusutil.CreateFolderChannel
}

type worker struct {
	quitChannel         chan struct{}
	errorChannel        chan error
	deleteFilesChannel  cirrusutil.DeleteFilesChannel
	moveFileChannel     cirrusutil.MoveFileChannel
	uploadFilesChannel  cirrusutil.UploadFilesChannel
	createFolderChannel cirrusutil.CreateFolderChannel
}

func NewWorker() Worker {
	return &worker{
		quitChannel:         make(chan struct{}),
		errorChannel:        make(chan error),
		deleteFilesChannel:  make(cirrusutil.DeleteFilesChannel),
		moveFileChannel:     make(cirrusutil.MoveFileChannel),
		uploadFilesChannel:  make(cirrusutil.UploadFilesChannel),
		createFolderChannel: make(cirrusutil.CreateFolderChannel),
	}
}

func (w *worker) Process() error {
	for {
		select {
		case deleteReq := <-w.deleteFilesChannel:
			if _, err := cirrusutil.DeleteFiles(deleteReq); err != nil {
				w.errorChannel <- err
			}
		case moveReq := <-w.moveFileChannel:
			if _, err := cirrusutil.MoveFile(moveReq); err != nil {
				w.errorChannel <- err
			}
		case uploadReq := <-w.uploadFilesChannel:
			if _, err := cirrusutil.UploadFiles(uploadReq); err != nil {
				w.errorChannel <- err
			}
		case createFolderReq := <-w.createFolderChannel:
			if _, err := cirrusutil.CreateFolder(createFolderReq); err != nil {
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

func (w *worker) GetDeleteFilesChannel() cirrusutil.DeleteFilesChannel {
	return w.deleteFilesChannel
}

func (w *worker) GetMoveFileChannel() cirrusutil.MoveFileChannel {
	return w.moveFileChannel
}

func (w *worker) GetUploadFilesChannel() cirrusutil.UploadFilesChannel {
	return w.uploadFilesChannel
}

func (w *worker) GetCreateFolderChannel() cirrusutil.CreateFolderChannel {
	return w.createFolderChannel
}
