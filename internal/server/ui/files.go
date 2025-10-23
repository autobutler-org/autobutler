package ui

import (
	"autobutler/internal/server/ui/components/file_explorer"
	"autobutler/internal/server/ui/components/file_explorer/file_viewer/docx_viewer"
	"autobutler/internal/server/ui/components/file_explorer/file_viewer/epub_viewer"
	"autobutler/internal/server/ui/components/file_explorer/file_viewer/image_viewer"
	"autobutler/internal/server/ui/components/file_explorer/file_viewer/pdf_viewer"
	"autobutler/internal/server/ui/components/file_explorer/file_viewer/text_viewer"
	"autobutler/internal/server/ui/components/file_explorer/file_viewer/unsupported_viewer"
	"autobutler/internal/server/ui/components/file_explorer/file_viewer/video_viewer"
	"autobutler/internal/server/ui/types"
	"autobutler/internal/server/ui/views"
	"autobutler/pkg/util"
	"html"
	"path/filepath"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

func SetupFileRoutes(router *gin.Engine) {
	setupFileView(router)
	setupComponentRoutes(router)
}

func setupFileView(router *gin.Engine) {
	uiRoute(router, "/files", func(c *gin.Context) {
		view := c.Query("view")
		if view == "" {
			view = "list"
		}

		// If this is an htmx request, return just the view content
		if c.GetHeader("HX-Request") == "true" {
			RenderFileExplorerViewContent(c, "", view)
			return
		}

		if err := views.Files(types.NewPageState().WithView(view)).Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
		c.Status(200)
	})
	uiRoute(router, "/files/*rootDir", func(c *gin.Context) {
		rootDir := c.Param("rootDir")
		view := c.Query("view")
		if view == "" {
			view = "list"
		}

		// If this is an htmx request, return just the view content
		if c.GetHeader("HX-Request") == "true" {
			RenderFileExplorerViewContent(c, rootDir, view)
			return
		}

		if err := views.Files(types.NewPageState().WithRootDir(rootDir).WithView(view)).Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
		c.Status(200)
	})
}

func setupComponentRoutes(router *gin.Engine) {
	setupComponentFileExplorer(router)
	setupComponentFileViewers(router)
}

func RenderFileExplorer(c *gin.Context, rootDir string) {
	renderFileExplorerHelper(c, rootDir, false)
}

func RenderFileExplorerViewContent(c *gin.Context, rootDir string, view string) {
	renderFileExplorerHelper(c, rootDir, true, view)
}

func renderFileExplorerHelper(c *gin.Context, rootDir string, viewContentOnly bool, view ...string) {
	fullPathDir := ""
	if rootDir == "" {
		fullPathDir = util.GetFilesDir()
	} else {
		fullPathDir = filepath.Join(util.GetFilesDir(), rootDir)
	}
	files, err := util.StatFilesInDir(fullPathDir)
	if err != nil {
		c.Writer.WriteString(`<span class="text-red-500">Failed to load files: ` + html.EscapeString(err.Error()) + `</span>`)
		return
	}

	viewStr := c.Query("view")
	if len(view) > 0 && view[0] != "" {
		viewStr = view[0]
	}
	if viewStr == "" {
		viewStr = "list"
	}

	var component templ.Component
	if viewContentOnly {
		component = file_explorer.ViewContent(types.NewPageState().WithRootDir(rootDir), files, viewStr)
	} else {
		component = file_explorer.Component(types.NewPageState().WithRootDir(rootDir), files, viewStr)
	}

	if err := component.Render(c.Request.Context(), c.Writer); err != nil {
		c.Status(500)
		return
	}
}

func setupComponentFileExplorer(router *gin.Engine) {
	uiRoute(router, "/components/files/explorer/*fileDir", func(c *gin.Context) {
		RenderFileExplorer(c, c.Param("fileDir"))
	})
}

func setupComponentFileViewers(router *gin.Engine) {
	uiRoute(router, "/components/files/viewer/files/*filePath", func(c *gin.Context) {
		filePath := c.Param("filePath")
		fileType := util.DetermineFileTypeFromPath(filePath)
		var viewer templ.Component
		switch fileType {
		case util.FileTypeImage:
			viewer = image_viewer.Component(filePath)
		case util.FileTypeVideo:
			viewer = video_viewer.Component(filePath)
		case util.FileTypePDF:
			viewer = pdf_viewer.Component(filePath)
		case util.FileTypeEpub:
			viewer = epub_viewer.Component(filePath)
		case util.FileTypeDocx:
			viewer = docx_viewer.Component(filePath)
		case util.FileTypeGeneric:
			viewer = text_viewer.Component(filePath)
		default:
			viewer = unsupported_viewer.Component(filePath)
		}
		if err := viewer.Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(500)
			return
		}
		c.Status(200)
	})
}
