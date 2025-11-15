package ui

import (
	"autobutler/internal/server/ui/components/photos"
	"autobutler/internal/server/ui/types"
	"autobutler/internal/server/ui/views"
	"autobutler/pkg/storage"
	"autobutler/pkg/util/fileutil"
	"autobutler/pkg/util/imageutil"
	"autobutler/pkg/util/serverutil"
	"strconv"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

func SetupPhotoRoutes(router *gin.Engine) {
	setupPhotoView(router)
	setupPhotoComponentRoutes(router)
}

func setupPhotoView(router *gin.Engine) {
	serverutil.UiRoute(router, "/photos", func(c *gin.Context) templ.Component {
		// Get storage summary for the storage bar component
		detector := storage.NewDetector()
		devices, err := detector.DetectDevices()
		var summary storage.Summary
		if err == nil && len(devices) > 0 {
			summary = detector.CalculateSummary(devices)
		} else {
			// Provide empty summary if detection fails
			summary = storage.Summary{}
		}

		return views.Photos(types.NewPageState(), summary)
	})
	serverutil.UiRoute(router, "/photos/*rootDir", func(c *gin.Context) templ.Component {
		rootDir := c.Param("rootDir")
		// Get storage summary for the storage bar component
		detector := storage.NewDetector()
		devices, err := detector.DetectDevices()
		var summary storage.Summary
		if err == nil && len(devices) > 0 {
			summary = detector.CalculateSummary(devices)
		} else {
			// Provide empty summary if detection fails
			summary = storage.Summary{}
		}

		return views.Photos(types.NewPageState().WithRootDir(rootDir), summary)
	})
}

func setupPhotoComponentRoutes(router *gin.Engine) {
	// Endpoint for infinite scroll pagination
	serverutil.UiRoute(router, "/components/photos/grid", func(c *gin.Context) templ.Component {
		pageStr := c.Query("page")
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}

		// DEBUG LOG
		println("🔍 SERVER: Photo grid request - Page:", page)

		// Get all photos
		photoFiles, err := imageutil.FindAllPhotosRecursively(fileutil.GetFilesDir())
		if err != nil {
			return nil
		}

		totalPhotos := len(photoFiles)
		pageSize := 50
		startIdx := (page - 1) * pageSize
		endIdx := startIdx + pageSize

		println("📊 SERVER: Total photos:", totalPhotos, "StartIdx:", startIdx, "EndIdx:", endIdx)

		if startIdx >= totalPhotos {
			return nil
		}

		if endIdx > totalPhotos {
			endIdx = totalPhotos
		}

		pagePhotos := photoFiles[startIdx:endIdx]
		println("✅ SERVER: Rendering", len(pagePhotos), "photos for page", page)

		// Set status before rendering
		c.Status(200)

		// Render the page component
		return photos.PhotoGridPage(types.NewPageState(), pagePhotos, page, totalPhotos)
	})
}
