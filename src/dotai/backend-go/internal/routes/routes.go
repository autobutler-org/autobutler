package routes

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"dotai-go-backend/internal/routes/profile"
	"dotai-go-backend/internal/routes/purchases"
	"dotai-go-backend/internal/database"
	"context"
)

func SetupRoutes(router *gin.Engine) {
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
	SetupRoutes(router)

	if err := router.Run(":8080"); err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}
	return nil
}
