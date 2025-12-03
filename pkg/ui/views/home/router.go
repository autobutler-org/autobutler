package view_home

import (
	"autobutler/pkg/ui/types"
	"autobutler/pkg/util/fileutil"
	"autobutler/pkg/util/serverutil"

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
		detector := fileutil.NewDetector()
		devices, err := detector.DetectDevices()
		var summary fileutil.Summary
		if err == nil && len(devices) > 0 {
			summary = fileutil.CalculateSummary(devices)
		} else {
			// Provide empty summary if detection fails
			summary = fileutil.Summary{}
		}

		return Home(types.NewPageState(), summary)
	},
)
