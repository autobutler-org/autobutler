package v1_migration

import (
	"autobutler/pkg/util/serverutil"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

type StartImportRequest struct {
	Services struct {
		Photos   bool `json:"photos"`
		Drive    bool `json:"drive"`
		Contacts bool `json:"contacts"`
		Calendar bool `json:"calendar"`
	} `json:"services"`
}

var googleStartRoute = serverutil.ApiRoute(
	"POST", "/migration/google/start", func(c *gin.Context) *serverutil.Response {
		var req StartImportRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.BadRequest(fmt.Errorf("invalid request: %w", err))
		}

		// TODO: Implement the actual Google Takeout flow:
		// 1. Use Google Takeout API to request export
		// 2. Create a background job to poll for completion
		// 3. When ready, download zip files
		// 4. POST files to /api/v1/cirrus endpoint
		// 5. Return job/export ID for tracking

		// For now, return a placeholder response
		exportId := fmt.Sprintf("export-%d", time.Now().Unix())

		return serverutil.Ok().WithData(gin.H{
			"exportId": exportId,
			"message":  "Export request submitted to Google Takeout",
			"services": req.Services,
		})
	},
)
