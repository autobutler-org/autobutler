package v1_networking

import "autobutler/pkg/util/serverutil"

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		getNodeStatusRoute,
		getNodeMetricsRoute,
		getNodeDiagnosticsRoute,
		getPrivacyFeaturesRoute,
		updatePrivacyFeaturesRoute,
		getConnectionInfoRoute,
	}
}
