package v1_photos

import (
	"github.com/autobutler-org/autobutler/pkg/util/photoutil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

type PhotoJSON struct {
	RelPath  string `json:"relPath"`
	FileName string `json:"fileName"`
	Size     int64  `json:"size"`
	MTime    int64  `json:"mtime"`
}

// listPhotos godoc
// @Summary List photos
// @Description Finds all photos in cirrus
// @Tags photos
// @Produce json
// @Success 200 {array} PhotoJSON
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /photos [get]
func listPhotos(c *gin.Context) *serverutil.Response {
	rootDir := storageutil.GetCirrusDir()
	photos, err := photoutil.FindAllPhotosRecursively(rootDir)
	if err != nil {
		return serverutil.InternalServerError(err)
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

var listPhotosRoute = serverutil.ApiRoute(
	"GET", "/photos", listPhotos,
)
