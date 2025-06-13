package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello, World!",
		})
	})

	if err := router.Run(":8080"); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}