package v1

import (
	"autobutler/pkg/util"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"autobutler/internal/server/ui/components/file_explorer/load"

	"github.com/gin-gonic/gin"
)

func SetupFilesRoutes(apiV1Group *gin.RouterGroup) {
	uploadFileRoute(apiV1Group)
}

func uploadFileRoute(apiV1Group *gin.RouterGroup) {
	apiRoute(apiV1Group, "POST", "/files/upload/*rootDir", func(c *gin.Context) {
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
