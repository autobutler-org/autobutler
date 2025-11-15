package v1

import (
	"autobutler/pkg/api"
	"autobutler/pkg/util/fileutil"
	"fmt"
	"html"
	"path/filepath"

	"autobutler/internal/quill"
	"autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

func SetupDocRoutes(apiV1Group *gin.RouterGroup) {
	saveDocRoute(apiV1Group)
}

func saveDocRoute(apiV1Group *gin.RouterGroup) {
	serverutil.ApiRoute(apiV1Group, "POST", "/docs/*filePath", func(c *gin.Context) *api.Response {
		filePath := c.Param("filePath")
		fileType := fileutil.DetermineFileTypeFromPath(filePath)
		switch fileType {
		case fileutil.FileTypeDocx:
			fmt.Println("Saving DOCX file:", filePath)
			var delta quill.Delta
			if err := c.BindJSON(&delta); err != nil {
				return api.NewResponse().WithStatusCode(400).WithData(`<span class="text-red-500">Failed to parse delta: ` + html.EscapeString(err.Error()) + `</span>`)
			}
			fullPath := filepath.Join(fileutil.GetFilesDir(), filePath)
			if err := delta.SaveDocxFile(fullPath); err != nil {
				return api.NewResponse().WithStatusCode(500).WithData(`<span class="text-red-500">Failed to save DOCX file: ` + html.EscapeString(err.Error()) + `</span>`)
			}
			return api.Ok()
		default:
			return api.NewResponse().WithStatusCode(400).WithData(`<span class="text-red-500">Unsupported file type for saving a doc: ` + html.EscapeString(string(fileType)) + `</span>`)
		}
	})
}
