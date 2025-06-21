package server

import (
	"fmt"
	"time"

	"autobutler/internal/llm"
	"autobutler/internal/update"
	"autobutler/pkg/util"
	"autobutler/ui/components/chat"
	"autobutler/ui/views"

	"github.com/gin-gonic/gin"
)

func setupRoutes(router *gin.Engine) {
	// STATIC FILES
	router.Static("/public", "./public")
	// UI ROUTES
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

	// API ROUTES
	apiV1Route(router, "GET", "/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})
	apiV1Route(router, "GET", "/chat", func(c *gin.Context) {
		isHtml := c.GetHeader("Accept") == "text/html"
		prompt := c.Query("prompt")
		response, err := llm.RemoteLLMRequest(prompt)
		if err != nil {
			if isHtml {
				messageComponent := chat.Message(llm.ErrorChatMessage())
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
			messageComponent := chat.Message(response.ToChatMessage())
			if err := messageComponent.Render(c.Request.Context(), c.Writer); err != nil {
				c.Status(500)
				return
			}
		} else {
			c.JSON(200, response)
		}
	})
	apiV1Route(router, "POST", "/update", func(c *gin.Context) {
		var r update.UpdateRequest
		if err := c.BindJSON(&r); err != nil {
			c.JSON(400, gin.H{
				"error": "Invalid request body",
			})
			return
		}
		if err := update.Update(r.Version); err != nil {
			c.JSON(500, gin.H{
				"error": err.Error(),
			})
			return
		}
		go update.RestartAutobutler(2 * time.Second)
		c.JSON(200, gin.H{
			"message": "Update successful, Autobutler will restart.",
		})
	})
}


func apiV1Route(router *gin.Engine, method string, path string, handler func(c *gin.Context)) gin.IRoutes {
	path = util.TrimLeading(path, '/')
	route := fmt.Sprintf("/api/v1/%s", path)
	switch method {
	case "GET": {
		return router.GET(route, handler)
	}
	case "POST": {
		return router.POST(route, handler)
	}
	case "PUT": {
		return router.PUT(route, handler)
	}
	case "DELETE": {
		return router.DELETE(route, handler)
	}
	default: {
		panic(fmt.Sprintf("Unsupported HTTP method: %s", method))
	}
	}
}

func uiRoute(router *gin.Engine, path string, handler func(c *gin.Context)) gin.IRoutes {
	path = util.TrimLeading(path, '/')
	route := fmt.Sprintf("/%s", path)
	return router.GET(route, handler)
}
