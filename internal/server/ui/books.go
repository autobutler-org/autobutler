package ui

import (
	"autobutler/internal/server/ui/types"
	"autobutler/internal/server/ui/views"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func SetupBookRoutes(router *gin.Engine) {
	setupBooksView(router)
	setupBookReaderView(router)
}

func setupBooksView(router *gin.Engine) {
	uiRoute(router, "/books", func(c *gin.Context) {
		if err := views.Books(types.NewPageState()).Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
		c.Status(200)
	})
}

func setupBookReaderView(router *gin.Engine) {
	uiRoute(router, "/books/reader", func(c *gin.Context) {
		// Get the book path from query parameter
		bookPath := c.Query("path")
		if bookPath == "" {
			c.String(400, "Missing path parameter")
			return
		}

		// Clean the path to prevent directory traversal
		bookPath = filepath.Clean(bookPath)

		if err := views.BookReader(bookPath).Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(500)
			return
		}
		c.Status(200)
	})
}
