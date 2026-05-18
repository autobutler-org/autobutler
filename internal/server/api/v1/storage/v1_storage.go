package v1_storage

import "github.com/autobutler-org/autobutler/pkg/util/serverutil"

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		disableUsbStorageDeviceRoute,
		enableUsbStorageDeviceRoute,
		getDeviceStatusBySerialRoute,
		backupToDeviceRoute,
		listDeviceStatusesRoute,
		renameDeviceRoute,
		setDeviceRoleRoute,
		startSnapshotBackupRoute,
		getSnapshotBackupStatusRoute,
		verifySnapshotBackupRoute,
	}
}
