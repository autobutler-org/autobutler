package v0_health

import (
	"github.com/autobutler-org/quark/pkg/util/healthutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
)

func NewRouter(collector *healthutil.Collector) serverutil.Router {
	return &router{collector: collector}
}
