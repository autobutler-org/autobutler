package v0_health

import (
	"github.com/autobutler-org/quark/pkg/util/healthutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
)

type router struct {
	collector *healthutil.Collector
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		r.getHealthRoute(),
	}
}
