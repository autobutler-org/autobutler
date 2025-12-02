package view_health

import (
	"autobutler/pkg/ui/types"
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
		healthRoute,
	}
}

var healthRoute = serverutil.UiRoute(
	"/health", func(c *gin.Context) templ.Component {
		return Health(types.NewPageState())
	},
)
