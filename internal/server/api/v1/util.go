package v1

import (
	"autobutler/pkg/util"
	"fmt"

	"github.com/gin-gonic/gin"
)

func apiRoute(router *gin.RouterGroup, method string, route string, handler func(c *gin.Context)) gin.IRoutes {
	route = util.TrimLeading(route, '/')
	switch method {
	case "GET":
		{
			return router.GET(route, handler)
		}
	case "POST":
		{
			return router.POST(route, handler)
		}
	case "PUT":
		{
			return router.PUT(route, handler)
		}
	case "DELETE":
		{
			return router.DELETE(route, handler)
		}
	default:
		{
			panic(fmt.Sprintf("Unsupported HTTP method: %s", method))
		}
	}
}
