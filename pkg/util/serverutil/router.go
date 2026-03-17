package serverutil

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Route struct {
	Path    string
	Method  string
	Handler gin.HandlerFunc
}

type Router interface {
	Routes() []*Route
}

func RegisterRouterWithGroup(group *gin.RouterGroup, router Router) {
	for _, route := range router.Routes() {
		group.Handle(route.Method, route.Path, route.Handler)
	}
}

func RegisterRouter(engine *gin.Engine, router Router) {
	for _, route := range router.Routes() {
		engine.Handle(route.Method, route.Path, route.Handler)
	}
}

func NewRoute(method, path string, handler gin.HandlerFunc) *Route {
	return &Route{
		Method:  method,
		Path:    path,
		Handler: handler,
	}
}

func ApiRoute(method, path string, handler func(c *gin.Context) *Response) *Route {
	return &Route{
		Method:  method,
		Path:    path,
		Handler: WrapApiRoute(handler),
	}
}

func WrapApiRoute(handler func(c *gin.Context) *Response) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := handler(c)
		// If handler wrote the response itself and returned nil, just return.
		if resp == nil {
			return
		}
		if resp.Data == nil && resp.Error == nil {
			c.Status(resp.StatusCode)
			return
		}
		// If the error wraps an HttpError, its status code takes precedence over
		// whatever status code the handler passed in. This means service functions
		// can signal HTTP semantics with fmt.Errorf("...: %w", httpErr) and the
		// HTTP layer will pick it up automatically.
		if resp.Error != nil {
			var httpErr *HttpError
			if errors.As(resp.Error, &httpErr) {
				resp.StatusCode = httpErr.StatusCode
			}
		}
		switch resp.ContentType {
		case ContentTypeHTML:
			if resp.Error != nil {
				c.String(resp.StatusCode, resp.Error.Error())
			} else {
				c.String(resp.StatusCode, "%v", resp.Data)
			}
		case ContentTypeJSON:
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
