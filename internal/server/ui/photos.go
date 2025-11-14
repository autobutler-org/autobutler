package ui

import (
	"autobutler/internal/server/ui/components/photos"
	"autobutler/internal/server/ui/types"
	"autobutler/internal/server/ui/views"
	"autobutler/pkg/storage"
	"autobutler/pkg/util"
	"strconv"

	"github.com/gin-gonic/gin"
)

func SetupPhotoRoutes(router *gin.Engine) {
	setupPhotoView(router)
	setupPhotoComponentRoutes(router)
}

func setupPhotoView(router *gin.Engine) {
	uiRoute(router, "/photos", func(c *gin.Context) {
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

		if err := views.Photos(types.NewPageState(), summary).Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
		c.Status(200)
	})
	uiRoute(router, "/photos/*rootDir", func(c *gin.Context) {
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

		if err := views.Photos(types.NewPageState().WithRootDir(rootDir), summary).Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
		c.Status(200)
	})
}

func setupPhotoComponentRoutes(router *gin.Engine) {
	// Endpoint for infinite scroll pagination
	uiRoute(router, "/components/photos/grid", func(c *gin.Context) {
		pageStr := c.Query("page")
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}

		// DEBUG LOG
		println("🔍 SERVER: Photo grid request - Page:", page)

		// Get all photos
		photoFiles, err := util.FindAllPhotosRecursively(util.GetFilesDir())
		if err != nil {
			println("❌ SERVER: Error loading photos:", err.Error())
			c.Writer.WriteString(`<div class="error-text">Error loading photos</div>`)
			return
		}

		totalPhotos := len(photoFiles)
		pageSize := 50
		startIdx := (page - 1) * pageSize
		endIdx := startIdx + pageSize

		println("📊 SERVER: Total photos:", totalPhotos, "StartIdx:", startIdx, "EndIdx:", endIdx)

		// Check bounds
		if startIdx >= totalPhotos {
			// No more photos
			println("⚠️  SERVER: No more photos to load (startIdx >= totalPhotos)")
			c.Status(200)
			return
		}

		if endIdx > totalPhotos {
			endIdx = totalPhotos
		}

		pagePhotos := photoFiles[startIdx:endIdx]
		println("✅ SERVER: Rendering", len(pagePhotos), "photos for page", page)

		// Set status before rendering
		c.Status(200)

		// Render the page component
		if err := photos.PhotoGridPage(types.NewPageState(), pagePhotos, page, totalPhotos).Render(c.Request.Context(), c.Writer); err != nil {
			println("❌ SERVER: Error rendering template:", err.Error())
			return
		}
	})
}
