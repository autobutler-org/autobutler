package v1_networking

import (
	"autobutler/pkg/networking"
	"autobutler/pkg/util/serverutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

var getNodeStatusRoute = serverutil.ApiRoute(
	"GET", "/networking/status", func(c *gin.Context) *serverutil.Response {
		node := getNodeFromContext(c)
		if node == nil {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusOK).
				WithData(gin.H{
					"name":            "autobutler.local",
					"local_ip":        "127.0.0.1",
					"ipv6":            "::1",
					"tailnet_ip":      "",
					"status":          "offline",
					"uptime":          0,
					"throughput_down": 0.0,
					"throughput_up":   0.0,
					"latency":         0.0,
					"configured":      false,
				})
		}

		info := node.GetInfo()
		metrics := node.GetMetrics()

		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusOK).
			WithData(gin.H{
				"name":            info.Name,
				"local_ip":        info.LocalIP,
				"ipv6":            info.IPv6,
				"tailnet_ip":      info.TailnetIP,
				"status":          info.Status,
				"uptime":          info.Uptime,
				"throughput_down": metrics.ThroughputDown,
				"throughput_up":   metrics.ThroughputUp,
				"latency":         metrics.Latency,
				"configured":      true,
			})
	},
)

func getNodeFromContext(c *gin.Context) *networking.Node {
	node, exists := c.Get("networking_node")
	if !exists {
		return nil
	}
	return node.(*networking.Node)
}
