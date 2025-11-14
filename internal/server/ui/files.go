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
	"autobutler/internal/serverutil"
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

func getViewFromRequest(c *gin.Context) string {
	// Check custom header first (from HTMX requests)
	if view := c.GetHeader("X-File-Explorer-View"); view != "" {
		return view
	}
	// Check cookie (synced from localStorage)
	if view, err := c.Cookie("fileExplorerView"); err == nil && view != "" {
		return view
	}
	// Fall back to query parameter (for direct URL access with ?view=)
	if view := c.Query("view"); view != "" {
		return view
	}
	// Default to list view
	return "list"
}

func setupFileView(router *gin.Engine) {
	serverutil.UiRoute(router, "/files", func(c *gin.Context) templ.Component {
		view := getViewFromRequest(c)

		// If this is an htmx request, return just the view content with OOB breadcrumb
		if c.GetHeader("HX-Request") == "true" {
			return GetFileExplorerViewContentWithBreadcrumb(c, "", view)
		}

		return views.Files(types.NewPageState().WithView(view))
	})
	serverutil.UiRoute(router, "/files/*rootDir", func(c *gin.Context) templ.Component {
		rootDir := c.Param("rootDir")
		view := getViewFromRequest(c)

		// If this is an htmx request, return just the view content with OOB breadcrumb
		if c.GetHeader("HX-Request") == "true" {
			return GetFileExplorerViewContentWithBreadcrumb(c, rootDir, view)
		}

		return views.Files(types.NewPageState().WithRootDir(rootDir).WithView(view))
	})
}

func setupComponentRoutes(router *gin.Engine) {
	setupComponentFileExplorer(router)
	setupComponentFileViewers(router)
}

func GetFileExplorer(c *gin.Context, rootDir string) templ.Component {
	return getFileExplorerComponent(c, rootDir, false)
}

func GetFileExplorerViewContent(c *gin.Context, rootDir string, view string) templ.Component {
	return getFileExplorerComponent(c, rootDir, true, view)
}

func GetFileExplorerViewContentWithBreadcrumb(c *gin.Context, rootDir string, view string) templ.Component {
	return getFileExplorerComponent(c, rootDir, true, view, true)
}

func getFileExplorerComponent(c *gin.Context, rootDir string, viewContentOnly bool, view ...any) templ.Component {
	fullPathDir := ""
	if rootDir == "" {
		fullPathDir = util.GetFilesDir()
	} else {
		fullPathDir = filepath.Join(util.GetFilesDir(), rootDir)
	}
	files, err := util.StatFilesInDir(fullPathDir)
	if err != nil {
		c.Writer.WriteString(`<span class="text-red-500">Failed to load files: ` + html.EscapeString(err.Error()) + `</span>`)
		return nil
	}

	viewStr := getViewFromRequest(c)
	withBreadcrumb := false

	// Parse variadic args: first is view string, second (optional) is withBreadcrumb bool
	if len(view) > 0 {
		if v, ok := view[0].(string); ok && v != "" {
			viewStr = v
		}
	}
	if len(view) > 1 {
		if wb, ok := view[1].(bool); ok {
			withBreadcrumb = wb
		}
	}

	var component templ.Component
	pageState := types.NewPageState().WithRootDir(rootDir).WithView(viewStr)
	if viewContentOnly {
		if withBreadcrumb {
			component = file_explorer.ViewContentWithBreadcrumb(pageState, files, viewStr)
		} else {
			component = file_explorer.ViewContent(pageState, files, viewStr)
		}
	} else {
		component = file_explorer.Component(pageState, files, viewStr)
	}
	return component
}

func setupComponentFileExplorer(router *gin.Engine) {
	serverutil.UiRoute(router, "/components/files/explorer/*fileDir", func(c *gin.Context) templ.Component {
		return GetFileExplorer(c, c.Param("fileDir"))
	})
}

func setupComponentFileViewers(router *gin.Engine) {
	serverutil.UiRoute(router, "/components/files/viewer/files/*filePath", func(c *gin.Context) templ.Component {
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
		return viewer
	})
}
