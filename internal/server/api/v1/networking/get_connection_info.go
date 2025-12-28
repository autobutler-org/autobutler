package v1_networking

import (
	"autobutler/pkg/util/serverutil"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

var getConnectionInfoRoute = serverutil.ApiRoute(
	"GET", "/networking/connection-info", func(c *gin.Context) *serverutil.Response {
		node := getNodeFromContext(c)
		if node == nil {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusServiceUnavailable).
				WithData(gin.H{
					"error": "Networking node not available",
				})
		}

		info := node.GetInfo()
		port := 8443

		httpsURL := fmt.Sprintf("https://%s:%d", info.Name, port)
		httpsIPURL := fmt.Sprintf("https://%s:%d", info.LocalIP, port)

		tailnetURL := ""
		if info.TailnetIP != "" {
			tailnetURL = fmt.Sprintf("https://%s:%d", info.TailnetIP, port)
		}

		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusOK).
			WithData(gin.H{
				"hostname":     info.Name,
				"local_ip":     info.LocalIP,
				"ipv6":         info.IPv6,
				"tailnet_ip":   info.TailnetIP,
				"port":         port,
				"https_url":    httpsURL,
				"https_ip_url": httpsIPURL,
				"tailnet_url":  tailnetURL,
				"instructions": []string{
					"1. Make sure you're on the same Wi-Fi or Ethernet network as this node.",
					fmt.Sprintf("2. Open this address in your browser: %s", httpsURL),
					fmt.Sprintf("3. Optional: If mDNS/local name resolution fails, use the IP address instead: %s", httpsIPURL),
				},
			})
	},
)
