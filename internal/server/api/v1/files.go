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

	"autobutler/internal/quill"
	"autobutler/internal/server/ui"
	"autobutler/internal/server/ui/components/file_explorer/load"

	"github.com/gin-gonic/gin"
)

func SetupFilesRoutes(apiV1Group *gin.RouterGroup) {
	deleteFileRoute(apiV1Group)
	downloadFileRoute(apiV1Group)
	newFolderRoute(apiV1Group)
	updateFileRoute(apiV1Group)
	uploadFileRoute(apiV1Group)
}

func deleteFileRoute(apiV1Group *gin.RouterGroup) {
	apiRoute(apiV1Group, "DELETE", "/files/*filePath", func(c *gin.Context) {
		filePath := c.Param("filePath")
		rootDir := util.GetFilesDir()
		fullPath := filepath.Join(rootDir, filePath)
		if err := os.RemoveAll(fullPath); err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		fileDir := filepath.Dir(filePath)
		ui.RenderFileExplorer(c, fileDir)
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
		ui.RenderFileExplorer(c, newDir)
	})
}

func updateFileRouteImpl(c *gin.Context, filePath string) {
	fileType := util.DetermineFileTypeFromPath(filePath)
	switch fileType {
	case util.FileTypeDocx:
		fmt.Println("Updating DOCX file:", filePath)
		var delta quill.Delta
		if err := c.BindJSON(&delta); err != nil {
			c.Writer.WriteString(`<span class="text-red-500">Failed to parse delta: ` + html.EscapeString(err.Error()) + `</span>`)
			c.Status(http.StatusBadRequest)
			return
		}
		fullPath := filepath.Join(util.GetFilesDir(), filePath)
		if err := delta.SaveDocxFile(fullPath); err != nil {
			c.Writer.WriteString(`<span class="text-red-500">Failed to save DOCX file: ` + html.EscapeString(err.Error()) + `</span>`)
			c.Status(http.StatusInternalServerError)
			return
		}
	default:
		panic(fmt.Sprintf("Unsupported file type for update: %s", fileType))
	}
}

func updateFileRoute(apiV1Group *gin.RouterGroup) {
	apiRoute(apiV1Group, "PUT", "/files/*filePath", func(c *gin.Context) {
		filePath := c.Param("filePath")
		updateFileRouteImpl(c, filePath)
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
	loadComponent := load.Component(rootDir)
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
