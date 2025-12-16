package v1_thumbnails

import (
	"autobutler/pkg/util/fileutil"
	"autobutler/pkg/util/photoutil"
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

var getThumbnailRoute = serverutil.ApiRoute(
	"GET", "/thumbnails/*filePath", func(c *gin.Context) *serverutil.Response {
		filePath := c.Param("filePath")
		filesDir := fileutil.GetCirrusDir()
		fullPath := filepath.Join(filesDir, filePath)

		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return serverutil.NewResponse().WithStatusCode(http.StatusNotFound)
		}

		result, err := photoutil.GenerateThumbnail(photoutil.GenerateThumbnailParams{
			FilePath: fullPath,
			Width:    thumbnailWidth,
			Height:   thumbnailHeight,
		})

		if err != nil {
			return serverutil.NewResponse().WithStatusCode(http.StatusInternalServerError).WithError(err)
		}

		ext := strings.ToLower(filepath.Ext(filePath))
		switch ext {
		case ".png":
			c.Header("Content-Type", "image/png")
			if err := png.Encode(c.Writer, result.Thumbnail); err != nil {
				return serverutil.NewResponse().WithStatusCode(http.StatusInternalServerError)
			}
		case ".jpg", ".jpeg":
			c.Header("Content-Type", "image/jpeg")
			if err := jpeg.Encode(c.Writer, result.Thumbnail, &jpeg.Options{Quality: 85}); err != nil {
				return serverutil.NewResponse().WithStatusCode(http.StatusInternalServerError)
			}
		default:
			// For other formats, try to encode as JPEG
			c.Header("Content-Type", fmt.Sprintf("image/%s", result.Format))
			if err := jpeg.Encode(c.Writer, result.Thumbnail, &jpeg.Options{Quality: 85}); err != nil {
				return serverutil.NewResponse().WithStatusCode(http.StatusInternalServerError)
			}
		}
		return serverutil.Ok()
	},
)
