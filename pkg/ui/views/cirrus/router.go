package view_cirrus

import (
	"autobutler/pkg/ui/components/file_explorer"
	"autobutler/pkg/ui/components/file_explorer/file_viewer/epub_viewer"
	"autobutler/pkg/ui/components/file_explorer/file_viewer/image_viewer"
	"autobutler/pkg/ui/components/file_explorer/file_viewer/pdf_viewer"
	"autobutler/pkg/ui/components/file_explorer/file_viewer/text_viewer"
	"autobutler/pkg/ui/components/file_explorer/file_viewer/unsupported_viewer"
	"autobutler/pkg/ui/components/file_explorer/file_viewer/video_viewer"
	"autobutler/pkg/ui/types"
	"autobutler/pkg/util/fileutil"
	"autobutler/pkg/util/serverutil"
	"html"
	"path/filepath"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

type router struct{}

func NewRouter() *router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		filesRoute,
		filesNestedRoute,
		fileExplorerComponentRoute,
		fileViewerComponentRoute,
	}
}

var filesRoute = serverutil.UiRoute(
	"/cirrus", func(c *gin.Context) templ.Component {
		view := getViewFromRequest(c)

		// If this is an htmx request, return just the view content with OOB breadcrumb
		if c.GetHeader("HX-Request") == "true" {
			return GetExplorerViewContentWithBreadcrumb(c, "", view)
		}

		return Files(types.NewPageState().WithView(view))
	},
)

var filesNestedRoute = serverutil.UiRoute(
	"/cirrus/*rootDir", func(c *gin.Context) templ.Component {
		rootDir := c.Param("rootDir")
		view := getViewFromRequest(c)

		// If this is an htmx request, return just the view content with OOB breadcrumb
		if c.GetHeader("HX-Request") == "true" {
			return GetExplorerViewContentWithBreadcrumb(c, rootDir, view)
		}

		return Files(types.NewPageState().WithRootDir(rootDir).WithView(view))
	},
)

var fileExplorerComponentRoute = serverutil.UiRoute(
	"/components/cirrus/explorer/*fileDir", func(c *gin.Context) templ.Component {
		return GetExplorer(c, c.Param("fileDir"))
	},
)

var fileViewerComponentRoute = serverutil.UiRoute(
	"/components/cirrus/viewer/cirrus/*filePath", func(c *gin.Context) templ.Component {
		filePath := c.Param("filePath")
		fileType := fileutil.DetermineFileTypeFromPath(filePath)
		var viewer templ.Component
		switch fileType {
		case fileutil.FileTypeImage:
			viewer = image_viewer.Component(filePath)
		case fileutil.FileTypeVideo:
			viewer = video_viewer.Component(filePath)
		case fileutil.FileTypePDF:
			viewer = pdf_viewer.Component(filePath)
		case fileutil.FileTypeEpub:
			viewer = epub_viewer.Component(filePath)
		case fileutil.FileTypeGeneric:
			viewer = text_viewer.Component(filePath)
		case fileutil.FileTypeDocx:
			viewer = unsupported_viewer.Component(filePath)
		default:
			viewer = unsupported_viewer.Component(filePath)
		}
		return viewer
	},
)

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

func GetExplorer(c *gin.Context, rootDir string) templ.Component {
	return getFileExplorerComponent(c, rootDir, false)
}

func GetExplorerViewContent(c *gin.Context, rootDir string, view string) templ.Component {
	return getFileExplorerComponent(c, rootDir, true, view)
}

func GetExplorerViewContentWithBreadcrumb(c *gin.Context, rootDir string, view string) templ.Component {
	return getFileExplorerComponent(c, rootDir, true, view, true)
}

func getFileExplorerComponent(c *gin.Context, rootDir string, viewContentOnly bool, view ...any) templ.Component {
	// Get all managed devices
	managedDevices, err := fileutil.GetManagedDevices()
	if err != nil {
		c.Writer.WriteString(`<span class="text-red-500">Failed to load managed devices: ` + html.EscapeString(err.Error()) + `</span>`)
		return nil
	}

	var files []*fileutil.DeviceFileInfo

	// If no managed devices exist, fall back to single default device
	if len(managedDevices) == 0 {
		fullPathDir := ""
		if rootDir == "" {
			fullPathDir = fileutil.GetCirrusDir()
		} else {
			fullPathDir = filepath.Join(fileutil.GetCirrusDir(), rootDir)
		}
		// Get device info for the default files directory
		deviceName, devicePath := fileutil.GetDeviceInfoForPath(fullPathDir)
		files, err = fileutil.StatFilesInDir(fullPathDir, deviceName, devicePath)
		if err != nil {
			c.Writer.WriteString(`<span class="text-red-500">Failed to load files: ` + html.EscapeString(err.Error()) + `</span>`)
			return nil
		}
	} else {
		// Build list of directories to scan across all devices
		var dirsToScan []fileutil.DirWithDevice
		for _, device := range managedDevices {
			dirPath := device.FilesDir
			if rootDir != "" {
				dirPath = filepath.Join(device.FilesDir, rootDir)
			}
			dirsToScan = append(dirsToScan, fileutil.DirWithDevice{
				Dir:        dirPath,
				DeviceName: device.Name,
				DevicePath: device.MountPoint,
			})
		}

		// Get unified file list across all devices
		files, err = fileutil.StatFilesInMultipleDirs(dirsToScan)
		if err != nil {
			c.Writer.WriteString(`<span class="text-red-500">Failed to load files: ` + html.EscapeString(err.Error()) + `</span>`)
			return nil
		}
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
