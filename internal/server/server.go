package server

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

func StartServer() error {
	router := gin.Default()
	// IMPORTANT: UseMiddleware MUST be called before setupRoutes
	useMiddleware(router)
	setupRoutes(router)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := router.Run(fmt.Sprintf(":%s", port)); err != nil {
		return err
	}

	return nil
}
