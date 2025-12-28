package v1_networking

import (
	"autobutler/pkg/util/serverutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

var getPrivacyFeaturesRoute = serverutil.ApiRoute(
	"GET", "/networking/features", func(c *gin.Context) *serverutil.Response {
		node := getNodeFromContext(c)
		if node == nil {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusOK).
				WithData(gin.H{
					"advertise_local": false,
					"remote_tunnel":   false,
					"usage_analytics": false,
					"configured":      false,
				})
		}

		features := node.GetFeatures()

		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusOK).
			WithData(gin.H{
				"advertise_local": features.AdvertiseLocal,
				"remote_tunnel":   features.RemoteTunnel,
				"usage_analytics": features.UsageAnalytics,
				"configured":      true,
			})
	},
)
