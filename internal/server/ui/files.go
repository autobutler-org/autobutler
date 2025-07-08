package ui

import (
	"autobutler/internal/server/ui/components/file_explorer"
	"autobutler/internal/server/ui/views"
	"autobutler/pkg/util"
	"html"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func SetupFileRoutes(router *gin.Engine) {
	setupFileView(router)
	setupComponentRoutes(router)
}

func setupFileView(router *gin.Engine) {
	uiRoute(router, "/files", func(c *gin.Context) {
		if err := views.Files("").Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
		c.Status(200)
	})
	uiRoute(router, "/files/*rootDir", func(c *gin.Context) {
		rootDir := c.Param("rootDir")
		if err := views.Files(rootDir).Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
		c.Status(200)
	})
}

func setupComponentRoutes(router *gin.Engine) {
	setupComponentFileExplorer(router)
}

func RenderFileExplorer(c *gin.Context, fileDir string) {
	isHtml := c.GetHeader("Accept") == "text/html"
	fullPathDir := ""
	if fileDir == "" {
		fullPathDir = util.GetFilesDir()
	} else {
		fullPathDir = filepath.Join(util.GetFilesDir(), fileDir)
	}
	files, err := util.StatFilesInDir(fullPathDir)
	if err != nil {
		if isHtml {
			c.Writer.WriteString(`<span class="text-red-500">Failed to load files: ` + html.EscapeString(err.Error()) + `</span>`)
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load files: " + err.Error()})
		}
		return
	}
	explorerComponent := file_explorer.Component(fileDir, files)
	if isHtml {
		if err := explorerComponent.Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(500)
			return
		}
	} else {
		c.JSON(200, gin.H{
			"message": "File explorer loaded successfully",
			"dir":     fileDir,
		})
	}
}

func setupComponentFileExplorer(router *gin.Engine) {
	uiRoute(router, "/components/files/explorer/*fileDir", func(c *gin.Context) {
		RenderFileExplorer(c, c.Param("fileDir"))
	})
}
