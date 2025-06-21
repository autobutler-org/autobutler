package server

import (
	"github.com/gin-gonic/gin"
)

func StartServer() error {
	router := gin.Default()
	// IMPORTANT: UseMiddleware MUST be called before SetupRoutes
	useMiddleware(router)
	setupRoutes(router)
	if err := router.Run(":8080"); err != nil {
		return err
	}

	return nil
}
