package routes

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"dotai-go-backend/internal/database"
	"dotai-go-backend/internal/routes/profile"
	"dotai-go-backend/internal/routes/purchases"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine, db *database.DB) {
	// Database middleware
	router.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	// Health check endpoint - this will call your Health method
	router.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		if err := db.Health(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  "database connection failed",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"database": "connected",
		})
	})

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello, World!",
		})
	})

	profile.SetupRoutes(router)
	purchases.SetupRoutes(router)
}

func StartServer(ctx context.Context, db *database.DB) error {
	router := gin.Default()
	SetupRoutes(router, db)

	if err := router.Run(":8080"); err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}
	return nil
}
