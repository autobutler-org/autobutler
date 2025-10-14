package v1

import (
	"autobutler/pkg/util"
	"fmt"
	"html"
	"net/http"
	"path/filepath"

	"autobutler/internal/quill"

	"github.com/gin-gonic/gin"
)

func SetupDocRoutes(apiV1Group *gin.RouterGroup) {
	saveDocRoute(apiV1Group)
}

func saveDocRoute(apiV1Group *gin.RouterGroup) {
	apiRoute(apiV1Group, "POST", "/docs/*filePath", func(c *gin.Context) {
		filePath := c.Param("filePath")
		fileType := util.DetermineFileTypeFromPath(filePath)
		switch fileType {
		case util.FileTypeDocx:
			fmt.Println("Saving DOCX file:", filePath)
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
			c.Writer.WriteString(`<span class="text-red-500">Unsupported file type for saving a doc: ` + html.EscapeString(string(fileType)) + `</span>`)
			c.Status(http.StatusBadRequest)
			return
		}
	})
}
