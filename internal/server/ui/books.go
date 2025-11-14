package ui

import (
	"autobutler/internal/server/ui/types"
	"autobutler/internal/server/ui/views"
	"autobutler/internal/serverutil"
	"path/filepath"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

func SetupBookRoutes(router *gin.Engine) {
	setupBooksView(router)
	setupBookReaderView(router)
}

func setupBooksView(router *gin.Engine) {
	serverutil.UiRoute(router, "/books", func(c *gin.Context) templ.Component {
		return views.Books(types.NewPageState())
	})
}

func setupBookReaderView(router *gin.Engine) {
	serverutil.UiRoute(router, "/books/reader", func(c *gin.Context) templ.Component {
		// Get the book path from query parameter
		bookPath := c.Query("path")
		if bookPath == "" {
			return nil
		}

		// Clean the path to prevent directory traversal
		bookPath = filepath.Clean(bookPath)

		return views.BookReader(bookPath)
	})
}
