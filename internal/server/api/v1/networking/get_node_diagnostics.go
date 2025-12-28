package v1_networking

import (
	"autobutler/pkg/util/serverutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

var getNodeDiagnosticsRoute = serverutil.ApiRoute(
	"GET", "/networking/diagnostics", func(c *gin.Context) *serverutil.Response {
		node := getNodeFromContext(c)
		if node == nil {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusOK).
				WithData(gin.H{
					"checks": []gin.H{
						{
							"name":        "Configuration",
							"description": "Networking node not configured. Set HEADSCALE_URL, HEADSCALE_AUTH_KEY, and NODE_HOSTNAME environment variables.",
							"status":      "warning",
							"status_text": "Not configured",
						},
					},
					"configured": false,
				})
		}

		diagnostics := node.GetDiagnostics(c.Request.Context())

		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusOK).
			WithData(gin.H{
				"checks":     diagnostics,
				"configured": true,
			})
	},
)
