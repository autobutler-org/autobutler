package v0_storage

import (
	"github.com/autobutler-org/quark/pkg/util/serverutil"
)

type router struct{}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		disableUsbStorageDeviceRoute,
		enableUsbStorageDeviceRoute,
		listDeviceStatusesRoute,
		renameDeviceRoute,
		setDeviceRoleRoute,
		startSnapshotBackupRoute,
		getSnapshotBackupStatusRoute,
		verifySnapshotBackupRoute,
	}
}
