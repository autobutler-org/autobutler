package v1_networking

import (
	"autobutler/pkg/util/serverutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

type updateFeaturesRequest struct {
	AdvertiseLocal *bool `json:"advertise_local"`
	RemoteTunnel   *bool `json:"remote_tunnel"`
	UsageAnalytics *bool `json:"usage_analytics"`
}

var updatePrivacyFeaturesRoute = serverutil.ApiRoute(
	"POST", "/networking/features", func(c *gin.Context) *serverutil.Response {
		node := getNodeFromContext(c)
		if node == nil {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusServiceUnavailable).
				WithData(gin.H{
					"error": "Networking node not available",
				})
		}

		var req updateFeaturesRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusBadRequest).
				WithData(gin.H{
					"error": "Invalid request body",
				})
		}

		features := node.GetFeatures()

		if req.AdvertiseLocal != nil {
			features.AdvertiseLocal = *req.AdvertiseLocal
		}
		if req.RemoteTunnel != nil {
			features.RemoteTunnel = *req.RemoteTunnel
		}
		if req.UsageAnalytics != nil {
			features.UsageAnalytics = *req.UsageAnalytics
		}

		if err := node.UpdateFeatures(features); err != nil {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusInternalServerError).
				WithData(gin.H{
					"error": "Failed to update features",
				})
		}

		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusOK).
			WithData(gin.H{
				"advertise_local": features.AdvertiseLocal,
				"remote_tunnel":   features.RemoteTunnel,
				"usage_analytics": features.UsageAnalytics,
			})
	},
)
