package server

import (
	"embed"
	"fmt"
	"html"
	"time"

	"autobutler/internal/llm"
	"autobutler/internal/update"
	"autobutler/pkg/util"
	"autobutler/ui/components/chat/load"
	"autobutler/ui/components/chat/message"
	"autobutler/ui/views"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

//go:embed public
var public embed.FS

func setupStaticRoutes(router *gin.Engine) error {
	staticFS, err := static.EmbedFolder(public, "public")
	if err != nil {
		return err
	}
	router.NoRoute(static.Serve("/public", staticFS))
	return nil
}

func setupUiRoutes(router *gin.Engine) {
	uiRoute(router, "/", func(c *gin.Context) {
		if err := views.Home().Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
		c.Status(200)
	})
	uiRoute(router, "/chat", func(c *gin.Context) {
		if err := views.Chat().Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
		c.Status(200)
	})
	uiRoute(router, "/files", func(c *gin.Context) {
		if err := views.Files().Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
		c.Status(200)
	})
}

func setupApiRoutes(router *gin.Engine) {
	apiV1Group := router.Group("/api/v1")
	apiRoute(apiV1Group, "GET", "/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})
	apiRoute(apiV1Group, "GET", "/user-chat", func(c *gin.Context) {
		isHtml := c.GetHeader("Accept") == "text/html"
		prompt := c.Query("prompt")
		msg := llm.UserChatMessage(prompt)
		if isHtml {
			messageComponent := message.Component(msg)
			if err := messageComponent.Render(c.Request.Context(), c.Writer); err != nil {
				c.Status(500)
				return
			}
			// Render a div with an hx-trigger="load"
			loadComponent := load.Component(prompt)
			if err := loadComponent.Render(c.Request.Context(), c.Writer); err != nil {
				c.Status(500)
				return
			}
		} else {
			c.JSON(500, gin.H{
				"error": "HTML rendering is required for this endpoint",
			})
		}
	})
	apiRoute(apiV1Group, "GET", "/ai-chat", func(c *gin.Context) {
		isHtml := c.GetHeader("Accept") == "text/html"
		prompt := c.Query("prompt")
		response, err := llm.RemoteLLMRequest(prompt)
		if err != nil {
			if isHtml {
				messageComponent := message.Component(llm.ErrorChatMessage(err))
				if err := messageComponent.Render(c.Request.Context(), c.Writer); err != nil {
					c.Status(500)
					return
				}
			} else {
				c.JSON(500, gin.H{
					"error": err.Error(),
				})
			}
			return
		}
		if isHtml {
			messageComponent := message.Component(llm.FromCompletionToChatMessage(*response))
			if err := messageComponent.Render(c.Request.Context(), c.Writer); err != nil {
				c.Status(500)
				return
			}
		} else {
			c.JSON(200, response)
		}
	})
	apiRoute(apiV1Group, "POST", "/update", func(c *gin.Context) {
		isHtml := c.GetHeader("Accept") == "text/html"
		var r update.UpdateRequest
		if err := c.BindJSON(&r); err != nil {
			if isHtml {
				c.Writer.WriteString(`<span class="text-red-500">Invalid request body</span>`)
			} else {
				c.JSON(400, gin.H{
					"error": "Invalid request body",
				})
			}
			return
		}
		if err := update.Update(r.Version); err != nil {
			if isHtml {
				c.Writer.WriteString(fmt.Sprintf(`<span class="text-red-500">%s</span>`, html.EscapeString(err.Error())))
			} else {
				c.JSON(500, gin.H{
					"error": err.Error(),
				})
			}
			return
		}
		go update.RestartAutobutler(2 * time.Second)
		if isHtml {
			c.Writer.WriteString(`<span class="text-green-500">Update successful, Autobutler will restart.</span>`)
		} else {
			c.JSON(200, gin.H{
				"message": "Update successful, Autobutler will restart.",
			})
		}
	})
}

func setupRoutes(router *gin.Engine) {
	setupStaticRoutes(router)
	setupUiRoutes(router)
	setupApiRoutes(router)
}

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

func uiRoute(router *gin.Engine, path string, handler func(c *gin.Context)) gin.IRoutes {
	path = util.TrimLeading(path, '/')
	route := fmt.Sprintf("/%s", path)
	return router.GET(route, handler)
}
