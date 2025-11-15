package v1

import (
	"autobutler/pkg/api"
	"autobutler/pkg/util/fileutil"
	"autobutler/pkg/util/imageutil"
	"autobutler/pkg/util/serverutil"
	"fmt"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	thumbnailWidth  = 400
	thumbnailHeight = 400
)

func SetupThumbnailRoutes(apiV1Group *gin.RouterGroup) {
	getThumbnailRoute(apiV1Group)
}

func getThumbnailRoute(apiV1Group *gin.RouterGroup) {
	serverutil.ApiRoute(apiV1Group, "GET", "/thumbnails/*filePath", func(c *gin.Context) *api.Response {
		filePath := c.Param("filePath")
		filesDir := fileutil.GetFilesDir()
		fullPath := filepath.Join(filesDir, filePath)

		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return api.NewResponse().WithStatusCode(http.StatusNotFound)
		}

		thumbnail, format, err := imageutil.ImageToThumbnail(fullPath, thumbnailWidth, thumbnailHeight)
		if err != nil {
			return api.NewResponse().WithStatusCode(http.StatusInternalServerError).WithError(err)
		}

		ext := strings.ToLower(filepath.Ext(filePath))
		switch ext {
		case ".png":
			c.Header("Content-Type", "image/png")
			if err := png.Encode(c.Writer, thumbnail); err != nil {
				return api.NewResponse().WithStatusCode(http.StatusInternalServerError)
			}
		case ".jpg", ".jpeg":
			c.Header("Content-Type", "image/jpeg")
			if err := jpeg.Encode(c.Writer, thumbnail, &jpeg.Options{Quality: 85}); err != nil {
				return api.NewResponse().WithStatusCode(http.StatusInternalServerError)
			}
		default:
			// For other formats, try to encode as JPEG
			c.Header("Content-Type", fmt.Sprintf("image/%s", format))
			if err := jpeg.Encode(c.Writer, thumbnail, &jpeg.Options{Quality: 85}); err != nil {
				return api.NewResponse().WithStatusCode(http.StatusInternalServerError)
			}
		}
		return api.Ok()
	})
}
