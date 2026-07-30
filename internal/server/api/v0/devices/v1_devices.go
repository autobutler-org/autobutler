package v0_devices

import "github.com/autobutler-org/autobutler/pkg/util/serverutil"

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		// Connected-device traffic log (existing)
		listDevicesRoute,
		deleteDeviceRoute,
		// Device registration and approval (new)
		registerDeviceRoute,         // POST  /devices/register         (public)
		checkDeviceAccessRoute,      // GET   /devices/access            (public)
		listRegisteredDevicesRoute,  // GET   /devices/registered
		pendingDeviceCountRoute,     // GET   /devices/registered/pending/count
		getRegisteredDeviceRoute,    // GET   /devices/registered/:id
		approveDeviceRoute,          // POST  /devices/registered/:id/approve
		revokeDeviceRoute,           // POST  /devices/registered/:id/revoke
		deleteRegisteredDeviceRoute, // DELETE /devices/registered/:id
	}
}
