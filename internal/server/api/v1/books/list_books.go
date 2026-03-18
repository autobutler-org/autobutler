package v1_books

import (
	"github.com/autobutler-org/autobutler/pkg/util/bookutil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

type BookJSON struct {
	RelPath  string `json:"relPath"`
	FileName string `json:"fileName"`
	Size     int64  `json:"size"`
	Type     string `json:"type"`
	MTime    int64  `json:"mtime"`
}

// listBooks godoc
// @Summary List books
// @Description Finds all books in the cirrus directory
// @Tags books
// @Produce json
// @Success 200 {array} BookJSON
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /books [get]
func listBooks(c *gin.Context) *serverutil.Response {
	rootDir := storageutil.GetCirrusDir()
	books, err := bookutil.FindAllBooksRecursively(rootDir)
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	result := make([]BookJSON, len(books))
	for i, book := range books {
		info := book.FileInfo
		fileType := storageutil.DetermineFileTypeFromPath(info.Name())
		result[i] = BookJSON{
			RelPath:  book.RelPath,
			FileName: info.Name(),
			Size:     info.Size(),
			MTime:    info.ModTime().Unix(),
			Type:     string(fileType),
		}
	}

	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(result)
}

var listBooksRoute = serverutil.ApiRoute(
	"GET", "/books", listBooks,
)
