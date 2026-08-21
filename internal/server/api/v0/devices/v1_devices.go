package v0_devices

import "github.com/autobutler-org/quark/pkg/util/serverutil"

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		listDevicesRoute,
		deleteDeviceRoute,
	}
}
