package v1_health

import (
	"github.com/autobutler-org/autobutler/pkg/botel/system"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
)

type router struct {
	collector *system.Collector
}

func NewRouter(collector *system.Collector) serverutil.Router {
	return &router{collector: collector}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		r.getHealthRoute(),
	}
}
