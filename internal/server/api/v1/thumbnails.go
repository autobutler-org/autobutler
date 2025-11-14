package v1

import (
	"autobutler/internal/serverutil"
	"autobutler/pkg/api"
	"autobutler/pkg/util"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nfnt/resize"
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
		filesDir := util.GetFilesDir()
		fullPath := filepath.Join(filesDir, filePath)

		// Check if file exists
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return api.NewResponse().WithStatusCode(http.StatusNotFound)
		}

		// Open the original image
		file, err := os.Open(fullPath)
		if err != nil {
			return api.NewResponse().WithStatusCode(http.StatusInternalServerError)
		}
		defer file.Close()

		// Decode the image
		img, format, err := image.Decode(file)
		if err != nil {
			// If we can't decode it, just serve the original file
			c.File(fullPath)
			return api.Ok()
		}

		// Generate thumbnail
		thumbnail := resize.Thumbnail(thumbnailWidth, thumbnailHeight, img, resize.Lanczos3)

		// Set appropriate content type
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
