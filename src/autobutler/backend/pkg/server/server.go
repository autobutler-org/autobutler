package server

import (
	"time"

	"github.com/exokomodo/exoflow/autobutler/backend/pkg/llm"
	"github.com/exokomodo/exoflow/autobutler/backend/pkg/update"
	"github.com/exokomodo/exoflow/autobutler/backend/pkg/views"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func UseMiddleware(router *gin.Engine) {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"POST", "GET", "PUT", "OPTIONS"}
	config.AllowHeaders = []string{"*"}
	config.ExposeHeaders = []string{"Content-Length"}
	config.AllowCredentials = true
	config.MaxAge = 12 * time.Hour
	router.Use(cors.New(config))
}

func SetupRoutes(router *gin.Engine) {
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})
	router.GET("/", func(c *gin.Context) {
		if err := views.Index().Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
		c.Status(200)
	})
	router.GET("/chat", func(c *gin.Context) {
		prompt := c.Query("prompt")
		response, err := llm.RemoteLLMRequest(prompt)
		if err != nil {
			c.JSON(500, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(200, response)
	})
	router.POST("/update", func(c *gin.Context) {
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

func StartServer() error {
	router := gin.Default()
	// IMPORTANT: UseMiddleware MUST be called before SetupRoutes
	UseMiddleware(router)
	SetupRoutes(router)
	if err := router.Run(":8080"); err != nil {
		return err
	}

	return nil
}
