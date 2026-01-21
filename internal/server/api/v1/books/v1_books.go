package v1_books

import (
	"autobutler/pkg/util/bookutil"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

type BookJSON struct {
	RelPath  string `json:"relPath"`
	FileName string `json:"fileName"`
	Size     int64  `json:"size"`
	Type     string `json:"type"`
	MTime    int64  `json:"mtime"`
}

func getBooksHandler(c *gin.Context) *serverutil.Response {
	rootDir := storageutil.GetCirrusDir()
	books, err := bookutil.FindAllBooksRecursively(rootDir)
	if err != nil {
		return serverutil.NewResponse().WithStatusCode(500).WithError(err)
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

var getBooksRoute = serverutil.ApiRoute(
	"GET", "/books", getBooksHandler,
)
