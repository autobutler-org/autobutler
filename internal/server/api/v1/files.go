package v1

import (
	"autobutler/pkg/util"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"autobutler/internal/server/ui"
	"autobutler/internal/server/ui/components/file_explorer/load"

	"github.com/gin-gonic/gin"
)

func SetupFilesRoutes(apiV1Group *gin.RouterGroup) {
	deleteFileRoute(apiV1Group)
	downloadFileRoute(apiV1Group)
	uploadFileRoute(apiV1Group)
}

func deleteFileRoute(apiV1Group *gin.RouterGroup) {
	apiRoute(apiV1Group, "DELETE", "/files/*filePath", func(c *gin.Context) {
		filePath := c.Param("filePath")
		rootDir := util.GetFilesDir()
		fullPath := filepath.Join(rootDir, filePath)
		if err := os.Remove(fullPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file: " + err.Error()})
			return
		}
		fileDir := filepath.Dir(filePath)
		ui.RenderFileExplorer(c, fileDir)
	})
}

func DownloadFile(c *gin.Context, filePath string) {
	rootDir := util.GetFilesDir()
	fullPath := filepath.Join(rootDir, filePath)

	file, err := os.Open(fullPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found: " + err.Error()})
		return
	}
	defer file.Close()

	c.Header("Content-Disposition", "attachment; filename="+filepath.Base(fullPath))
	c.Header("Content-Type", "application/octet-stream")
	c.File(fullPath)
}

func downloadFileRoute(apiV1Group *gin.RouterGroup) {
	apiRoute(apiV1Group, "GET", "/files/*filePath", func(c *gin.Context) {
		filePath := c.Param("filePath")
		DownloadFile(c, filePath)
	})
}

func uploadFileRoute(apiV1Group *gin.RouterGroup) {
	apiRoute(apiV1Group, "POST", "/files/*rootDir", func(c *gin.Context) {
		isHtml := c.GetHeader("Accept") == "text/html"
		rootDir := c.Param("rootDir")
		// Parse the multipart form with a max memory size
		err := c.Request.ParseMultipartForm(32 << 20)
		if err != nil {
			if isHtml {
				c.Writer.WriteString(`<span class="text-red-500">Failed to parse multipart form: ` + html.EscapeString(err.Error()) + `</span>`)
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form: " + err.Error()})
			}
			return
		}

		file, header, err := c.Request.FormFile("file")
		if err != nil {
			if isHtml {
				c.Writer.WriteString(`<span class="text-red-500">Failed to get file: ` + html.EscapeString(err.Error()) + `</span>`)
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file: " + err.Error()})
			}
			return
		}
		defer file.Close()

		fileDir := util.GetFilesDir()
		newFilePath := filepath.Join(fileDir, rootDir, header.Filename)
		newFile, err := os.Create(newFilePath)
		if err != nil {
			if isHtml {
				c.Writer.WriteString(`<span class="text-red-500">Failed to create file: ` + html.EscapeString(err.Error()) + `</span>`)
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create file: " + err.Error()})
			}
			return
		}
		defer newFile.Close()
		if _, err := io.Copy(newFile, file); err != nil {
			if isHtml {
				c.Writer.WriteString(`<span class="text-red-500">Failed to write file: ` + html.EscapeString(err.Error()) + `</span>`)
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write file: " + err.Error()})
			}
			return
		}
		if isHtml {
			loadComponent := load.Component(rootDir)
			if err := loadComponent.Render(c.Request.Context(), c.Writer); err != nil {
				c.Status(500)
				return
			}
		} else {
			c.JSON(200, gin.H{
				"message": "File uploaded successfully",
				"file":    header.Filename,
			})
		}
	})
}
