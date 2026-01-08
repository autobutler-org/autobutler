package v1_storage

import "autobutler/pkg/util/serverutil"

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		disableUsbStorageDeviceRoute,
		enableUsbStorageDeviceRoute,
		findUsbStorageDeviceRoute,
		getDeviceStatusBySerialRoute,
		getDeviceStatusesRoute,
		initializeManagedDeviceRoute,
		listUsbStorageDevicesRoute,
	}
}
