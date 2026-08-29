package workerutil

import "github.com/autobutler-org/quark/pkg/util/storageutil"

type worker struct {
	quitChannel           chan struct{}
	errorChannel          chan error
	backupToDeviceChannel storageutil.BackupToDeviceChannel
	storageService        *storageutil.StorageService
}
