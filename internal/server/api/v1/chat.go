package v1

import (
	"autobutler/internal/llm"
	"autobutler/internal/server/ui/components/chat/load"
	"autobutler/internal/server/ui/components/chat/message"

	"github.com/gin-gonic/gin"
)

func SetupChatRoutes(apiV1Group *gin.RouterGroup) {
	aiChatRoute(apiV1Group)
	userChatRoute(apiV1Group)
}

func aiChatRoute(apiV1Group *gin.RouterGroup) {
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
}

func userChatRoute(apiV1Group *gin.RouterGroup) {
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
}
