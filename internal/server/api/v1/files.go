package v1

import (
	"archive/zip"
	"autobutler/pkg/util"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"autobutler/internal/server/ui"
	"autobutler/internal/server/ui/components/file_explorer/load"
	"autobutler/internal/server/ui/types"

	"github.com/gin-gonic/gin"
)

func SetupFilesRoutes(apiV1Group *gin.RouterGroup) {
	deleteFilesRoute(apiV1Group)
	downloadFileRoute(apiV1Group)
	newFolderRoute(apiV1Group)
	moveFileRoute(apiV1Group)
	uploadFileRoute(apiV1Group)
}

func deleteFilesRoute(apiV1Group *gin.RouterGroup) {
	apiRoute(apiV1Group, "DELETE", "/files", func(c *gin.Context) {
		rootDir := c.Query("rootDir")
		filePaths := c.QueryArray("filePaths")
		fmt.Printf("Deleting multiple files: %s\n", filePaths)
		fileDir := util.GetFilesDir()
		for _, filePath := range filePaths {
			fullPath := filepath.Join(fileDir, rootDir, filePath)
			if err := os.RemoveAll(fullPath); err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}
		}
		// Always render the full file explorer (button targets #file-explorer)
		ui.RenderFileExplorer(c, rootDir)
	})
}

func DownloadFile(c *gin.Context, filePath string) {
	rootDir := util.GetFilesDir()
	fullPath := filepath.Join(rootDir, filePath)

	fileType := util.DetermineFileTypeFromPath(fullPath)

	if fileType == util.FileTypeFolder {
		zipWriter := zip.NewWriter(c.Writer)
		defer zipWriter.Close()
		dirFs := os.DirFS(fullPath)
		err := zipWriter.AddFS(dirFs)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", filepath.Base(fullPath)))
		c.Writer.Header().Set("Content-Type", "application/zip")
	} else {
		file, err := os.Open(fullPath)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		defer file.Close()

		disposition := "inline"
		contentType := "application/octet-stream"
		if fileType == util.FileTypePDF {
			disposition = "inline"
			contentType = "application/pdf"
		}
		c.Header("Content-Disposition", fmt.Sprintf("%s; filename=%s", disposition, filepath.Base(fullPath)))
		c.Header("Content-Type", contentType)
		c.File(fullPath)
	}
}

func downloadFileRoute(apiV1Group *gin.RouterGroup) {
	apiRoute(apiV1Group, "GET", "/files/*filePath", func(c *gin.Context) {
		filePath := c.Param("filePath")
		DownloadFile(c, filePath)
	})
}

func newFolderRoute(apiV1Group *gin.RouterGroup) {
	apiRoute(apiV1Group, "POST", "/folder/files/*folderDir", func(c *gin.Context) {
		folderDir := c.Param("folderDir")
		folderName := c.PostForm("folderName")
		rootDir := util.GetFilesDir()
		fullPath := filepath.Join(rootDir, folderDir, folderName)

		if err := os.MkdirAll(fullPath, 0755); err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		newDir := filepath.Join(folderDir, folderName)
		// Check if it's an HTMX request targeting just the content
		if c.GetHeader("HX-Request") == "true" {
			ui.RenderFileExplorerViewContent(c, newDir, "")
		} else {
			ui.RenderFileExplorer(c, newDir)
		}
	})
}

func moveFileRoute(apiV1Group *gin.RouterGroup) {
	apiRoute(apiV1Group, "PUT", "/files/*filePath", func(c *gin.Context) {
		filePath := c.Param("filePath")
		newFilePath := c.PostForm("newFilePath")
		filesDir := util.GetFilesDir()
		oldFullPath := filepath.Join(filesDir, filePath)
		newFullPath := filepath.Join(filesDir, newFilePath)

		newFullDir := filepath.Dir(newFullPath)
		if err := os.MkdirAll(newFullDir, 0755); err != nil {
			c.Writer.WriteString(`<span class="text-red-500">` + html.EscapeString(err.Error()) + `</span>`)
			c.Status(http.StatusInternalServerError)
			return
		}
		if err := os.Rename(oldFullPath, newFullPath); err != nil {
			c.Writer.WriteString(`<span class="text-red-500">` + html.EscapeString(err.Error()) + `</span>`)
			c.Status(http.StatusInternalServerError)
			return
		}
		newDir := filepath.Dir(newFilePath)
		if newDir == "." {
			newDir = ""
		}
		// Always render the full file explorer (JS function targets #file-explorer)
		ui.RenderFileExplorer(c, newDir)
	})
}

func uploadFileRouteImpl(c *gin.Context, rootDir string) {
	// Parse the multipart form with a max memory size
	err := c.Request.ParseMultipartForm(32 << 20)
	if err != nil {
		c.Writer.WriteString(`<span class="text-red-500">Failed to parse multipart form: ` + html.EscapeString(err.Error()) + `</span>`)
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.Writer.WriteString(`<span class="text-red-500">Failed to get file: ` + html.EscapeString(err.Error()) + `</span>`)
		return
	}
	fileHeaders := form.File["files"]
	for _, header := range fileHeaders {
		file, err := header.Open()
		if err != nil {
			c.Writer.WriteString(`<span class="text-red-500">Failed to open file: ` + html.EscapeString(err.Error()) + `</span>`)
			return
		}
		defer file.Close()

		fileDir := util.GetFilesDir()
		newFilePath := filepath.Join(fileDir, rootDir, header.Filename)
		newFile, err := os.Create(newFilePath)
		if err != nil {
			c.Writer.WriteString(`<span class="text-red-500">Failed to create file: ` + html.EscapeString(err.Error()) + `</span>`)
			return
		}
		defer newFile.Close()
		if _, err := io.Copy(newFile, file); err != nil {
			c.Writer.WriteString(`<span class="text-red-500">Failed to write file: ` + html.EscapeString(err.Error()) + `</span>`)
			return
		}
	}
	returnDir := form.Value["returnDir"]
	if len(returnDir) > 0 {
		rootDir = returnDir[0]
	}
	loadComponent := load.Component(types.NewPageState().WithRootDir(rootDir))
	if err := loadComponent.Render(c.Request.Context(), c.Writer); err != nil {
		c.Status(500)
		return
	}
}

func uploadFileRoute(apiV1Group *gin.RouterGroup) {
	apiRoute(apiV1Group, "POST", "/files", func(c *gin.Context) {
		uploadFileRouteImpl(c, "")
	})
	apiRoute(apiV1Group, "POST", "/files/*rootDir", func(c *gin.Context) {
		rootDir := c.Param("rootDir")
		uploadFileRouteImpl(c, rootDir)
	})
}
