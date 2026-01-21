package v1_photos

import (
	"autobutler/pkg/util/photoutil"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

type PhotoJSON struct {
	RelPath  string `json:"relPath"`
	FileName string `json:"fileName"`
	Size     int64  `json:"size"`
	MTime    int64  `json:"mtime"`
}

func getPhotosHandler(c *gin.Context) *serverutil.Response {
	rootDir := storageutil.GetCirrusDir()
	photos, err := photoutil.FindAllPhotosRecursively(rootDir)
	if err != nil {
		return serverutil.NewResponse().WithStatusCode(500).WithError(err)
	}

	result := make([]PhotoJSON, len(photos))
	for i, photo := range photos {
		info := photo.FileInfo
		result[i] = PhotoJSON{
			RelPath:  photo.RelPath,
			FileName: info.Name(),
			Size:     info.Size(),
			MTime:    info.ModTime().Unix(),
		}
	}

	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(result)
}

var getPhotosRoute = serverutil.ApiRoute(
	"GET", "/photos", getPhotosHandler,
)
