package view_home

import (
	"autobutler/pkg/ui/types"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

type router struct{}

func NewRouter() *router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		homeRoute,
	}
}

var homeRoute = serverutil.UiRoute(
	"/", func(c *gin.Context) templ.Component {
		// Get storage summary for the storage bar component
		detector := storageutil.NewDetector()
		devices, err := detector.DetectDevices()
		var summary storageutil.Summary
		if err == nil && len(devices) > 0 {
			summary = storageutil.CalculateSummary(devices)
		} else {
			// Provide empty summary if detection fails
			summary = storageutil.Summary{}
		}

		return Home(types.NewPageState(), summary)
	},
)
