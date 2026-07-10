package v0_metrics

import (
	"net/http"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

var listMetricsRoute = serverutil.NewRoute(
	"GET", "/metrics", func(c *gin.Context) {
		deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
		if !ok {
			c.String(http.StatusInternalServerError, "# Dependencies not found in context\n")
			return
		}

		metricsExporter := deps.MetricsExporter()
		if metricsExporter == nil {
			c.String(http.StatusServiceUnavailable, "# Metrics exporter not initialized\n")
			return
		}

		metrics, err := metricsExporter.PrometheusMetrics(c.Request.Context())
		if err != nil {
			c.String(http.StatusInternalServerError, "# Error generating metrics: %s\n", err.Error())
			return
		}

		c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		c.String(http.StatusOK, metrics)
	},
)
