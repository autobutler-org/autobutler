package v1_networking

import (
	"autobutler/pkg/util/serverutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

var getNodeMetricsRoute = serverutil.ApiRoute(
	"GET", "/networking/metrics", func(c *gin.Context) *serverutil.Response {
		node := getNodeFromContext(c)
		if node == nil {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusOK).
				WithData(gin.H{
					"active_clients":     0,
					"encrypted_sessions": 0,
					"total_sessions":     0,
					"blocked_requests":   0,
					"last_device_name":   "",
					"last_device_time":   "",
					"throughput_down":    0.0,
					"throughput_up":      0.0,
					"latency":            0.0,
					"uptime":             0.0,
					"configured":         false,
				})
		}

		metrics := node.GetMetrics()

		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusOK).
			WithData(gin.H{
				"active_clients":     metrics.ActiveClients,
				"encrypted_sessions": metrics.EncryptedSessions,
				"total_sessions":     metrics.TotalSessions,
				"blocked_requests":   metrics.BlockedRequests,
				"last_device_name":   metrics.LastDeviceName,
				"last_device_time":   metrics.LastDeviceTime,
				"throughput_down":    metrics.ThroughputDown,
				"throughput_up":      metrics.ThroughputUp,
				"latency":            metrics.Latency,
				"uptime":             metrics.Uptime.Seconds(),
				"configured":         true,
			})
	},
)
