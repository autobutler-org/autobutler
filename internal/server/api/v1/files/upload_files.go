package v1_files

import (
	"autobutler/pkg/ui/components/file_explorer/load"
	"autobutler/pkg/ui/types"
	"autobutler/pkg/util/fileutil"
	"autobutler/pkg/util/serverutil"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

var uploadRootFilesRoute = serverutil.ApiRoute(
	"POST", "/files", func(c *gin.Context) *serverutil.Response {
		uploadFilesImpl(c, "")
		return serverutil.Ok()
	},
)
var uploadNestedFilesRoutes = serverutil.ApiRoute(
	"POST", "/files/*rootDir", func(c *gin.Context) *serverutil.Response {
		rootDir := c.Param("rootDir")
		uploadFilesImpl(c, rootDir)
		return serverutil.Ok()
	},
)

func uploadFilesImpl(c *gin.Context, rootDir string) {
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

		fileDir := fileutil.GetFilesDir()
		newFilePath := filepath.Join(fileDir, rootDir, header.Filename)
		if _, err := os.Stat(newFilePath); err == nil {
			ext := filepath.Ext(header.Filename)
			name := header.Filename[:len(header.Filename)-len(ext)]
			i := 1
			for {
				newFileName := fmt.Sprintf("%s_(%d)%s", name, i, ext)
				newFilePath = filepath.Join(fileDir, rootDir, newFileName)
				if _, err := os.Stat(newFilePath); os.IsNotExist(err) {
					break
				}
				i++
			}
		}
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
