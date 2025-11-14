package serverutil

import (
	"autobutler/pkg/api"
	"autobutler/pkg/util"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

func wrapApiRoute(handler func(c *gin.Context) *api.Response) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := handler(c)
		if resp.Data == nil && resp.Error == nil {
			c.Status(resp.StatusCode)
			return
		}
		switch resp.ContentType {
		case api.ContentTypeHTML:
			if resp.Error != nil {
				c.String(resp.StatusCode, resp.Error.Error())
			} else {
				c.String(resp.StatusCode, "%v", resp.Data)
			}
		case api.ContentTypeJSON:
			if resp.Error != nil {
				c.JSON(resp.StatusCode, gin.H{"error": resp.Error.Error()})
			} else {
				c.JSON(resp.StatusCode, resp.Data)
			}
		default:
			c.String(http.StatusInternalServerError, "Unsupported content type")
		}
	}
}

func ApiRoute(router *gin.RouterGroup, method string, route string, handler func(c *gin.Context) *api.Response) gin.IRoutes {
	route = util.TrimLeading(route, '/')
	wrapped := wrapApiRoute(handler)
	switch method {
	case "GET":
		{

			return router.GET(route, wrapped)
		}
	case "POST":
		{
			return router.POST(route, wrapped)
		}
	case "PUT":
		{
			return router.PUT(route, wrapped)
		}
	case "DELETE":
		{
			return router.DELETE(route, wrapped)
		}
	default:
		{
			panic(fmt.Sprintf("Unsupported HTTP method: %s", method))
		}
	}
}

func wrapUiRoute(handler func(c *gin.Context) templ.Component) gin.HandlerFunc {
	return func(c *gin.Context) {
		wrapped := wrapApiRoute(func(c *gin.Context) *api.Response {
			return api.NewResponse().WithStatusCode(400)
		})
		component := handler(c)
		if component == nil {
			wrapped = wrapApiRoute(func(c *gin.Context) *api.Response {
				return api.NewResponse().WithStatusCode(400)
			})
		} else if err := component.Render(c.Request.Context(), c.Writer); err != nil {
			wrapped = wrapApiRoute(func(c *gin.Context) *api.Response {
				return api.NewResponse().WithStatusCode(400)
			})
		}
		wrapped(c)
	}
}

func UiRoute(router *gin.Engine, path string, handler func(c *gin.Context) templ.Component) gin.IRoutes {
	path = util.TrimLeading(path, '/')
	route := filepath.Join("/", path)
	wrapped := wrapUiRoute(handler)
	return router.GET(route, wrapped)
}
