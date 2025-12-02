package view_books

import (
	"autobutler/pkg/ui/types"
	"autobutler/pkg/util/serverutil"
	"path/filepath"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

type router struct{}

func NewRouter() *router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		booksRoute,
		bookReaderRoute,
	}
}

var booksRoute = serverutil.UiRoute(
	"/books", func(c *gin.Context) templ.Component {
		return Books(types.NewPageState())
	},
)

var bookReaderRoute = serverutil.UiRoute(
	"/books/reader", func(c *gin.Context) templ.Component {
		// Get the book path from query parameter
		bookPath := c.Query("path")
		if bookPath == "" {
			return nil
		}

		// Clean the path to prevent directory traversal
		bookPath = filepath.Clean(bookPath)

		return BookReader(bookPath)
	},
)
