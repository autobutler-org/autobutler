package v1_metrics

import (
	"autobutler/pkg/util/ctxutil"
	"autobutler/pkg/util/deputil"
	"net/http"

	"github.com/gin-gonic/gin"
)

func handleMetrics(c *gin.Context) {
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
}
