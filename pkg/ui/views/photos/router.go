package view_photos

import (
	"autobutler/pkg/ui/components/photos"
	"autobutler/pkg/ui/types"
	"autobutler/pkg/util/fileutil"
	"autobutler/pkg/util/photoutil"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"
	"strconv"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

type router struct{}

func NewRouter() *router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		photosRoute,
		photosNestedRoute,
	}
}

var photosRoute = serverutil.UiRoute(
	"/photos", func(c *gin.Context) templ.Component {
		// Get storage summary for the storage bar component
		detector := storageutil.NewDetector()
		devices, err := detector.DetectDevices()
		var summary storageutil.Summary
		if err == nil && len(devices) > 0 {
			summary = storageutil.CalculateSummary(devices)
		} else {
			// Provide empty summary if detection fails
			summary = storageutil.Summary{}
		}

		return Photos(types.NewPageState(), summary)
	},
)

var photosNestedRoute = serverutil.UiRoute(
	"/photos/*rootDir", func(c *gin.Context) templ.Component {
		rootDir := c.Param("rootDir")
		// Get storage summary for the storage bar component
		detector := storageutil.NewDetector()
		devices, err := detector.DetectDevices()
		var summary storageutil.Summary
		if err == nil && len(devices) > 0 {
			summary = storageutil.CalculateSummary(devices)
		} else {
			// Provide empty summary if detection fails
			summary = storageutil.Summary{}
		}

		return Photos(types.NewPageState().WithRootDir(rootDir), summary)
	},
)

var photoGridComponentRoute = serverutil.UiRoute(
	"/components/photos/grid", func(c *gin.Context) templ.Component {
		pageStr := c.Query("page")
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}

		// DEBUG LOG
		println("🔍 SERVER: Photo grid request - Page:", page)

		// Get all photos
		photoFiles, err := photoutil.FindAllPhotosRecursively(fileutil.GetFilesDir())
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
	},
)
