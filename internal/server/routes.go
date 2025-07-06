package server

import (
	"embed"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"autobutler/internal/llm"
	"autobutler/internal/update"
	"autobutler/pkg/util"
	"autobutler/ui/components/chat/load"
	"autobutler/ui/components/chat/message"
	"autobutler/ui/components/file_explorer"
	file_explorer_load "autobutler/ui/components/file_explorer/load"
	"autobutler/ui/views"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

//go:embed public
var public embed.FS

func setupStaticRoutes(router *gin.Engine) error {
	staticFS, err := static.EmbedFolder(public, "public")
	if err != nil {
		return err
	}
	router.NoRoute(static.Serve("/public", staticFS))
	return nil
}

func setupUiRoutes(router *gin.Engine) {
	uiRoute(router, "/", func(c *gin.Context) {
		if err := views.Home().Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
		c.Status(200)
	})
	uiRoute(router, "/chat", func(c *gin.Context) {
		if err := views.Chat().Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
		c.Status(200)
	})
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

func setupApiRoutes(router *gin.Engine) {
	apiV1Group := router.Group("/api/v1")
	apiRoute(apiV1Group, "GET", "/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})
	apiRoute(apiV1Group, "GET", "/user-chat", func(c *gin.Context) {
		isHtml := c.GetHeader("Accept") == "text/html"
		prompt := c.Query("prompt")
		msg := llm.UserChatMessage(prompt)
		if isHtml {
			messageComponent := message.Component(msg)
			if err := messageComponent.Render(c.Request.Context(), c.Writer); err != nil {
				c.Status(500)
				return
			}
			// Render a div with an hx-trigger="load"
			loadComponent := load.Component(prompt)
			if err := loadComponent.Render(c.Request.Context(), c.Writer); err != nil {
				c.Status(500)
				return
			}
		} else {
			c.JSON(500, gin.H{
				"error": "HTML rendering is required for this endpoint",
			})
		}
	})
	apiRoute(apiV1Group, "GET", "/ai-chat", func(c *gin.Context) {
		isHtml := c.GetHeader("Accept") == "text/html"
		prompt := c.Query("prompt")
		response, err := llm.RemoteLLMRequest(prompt)
		if err != nil {
			if isHtml {
				messageComponent := message.Component(llm.ErrorChatMessage(err))
				if err := messageComponent.Render(c.Request.Context(), c.Writer); err != nil {
					c.Status(500)
					return
				}
			} else {
				c.JSON(500, gin.H{
					"error": err.Error(),
				})
			}
			return
		}
		if isHtml {
			messageComponent := message.Component(llm.FromCompletionToChatMessage(*response))
			if err := messageComponent.Render(c.Request.Context(), c.Writer); err != nil {
				c.Status(500)
				return
			}
		} else {
			c.JSON(200, response)
		}
	})
	apiRoute(apiV1Group, "GET", "/files/explorer/*rootDir", func(c *gin.Context) {
		isHtml := c.GetHeader("Accept") == "text/html"
		rootDir := c.Param("rootDir")
		if rootDir == "" {
			rootDir = util.GetFilesDir()
		} else {
			rootDir = filepath.Join(util.GetFilesDir(), rootDir)
		}
		files, err := util.StatFilesInDir(rootDir)
		if err != nil {
			if isHtml {
				c.Writer.WriteString(`<span class="text-red-500">Failed to load files: ` + html.EscapeString(err.Error()) + `</span>`)
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load files: " + err.Error()})
			}
			return
		}
		loadComponent := file_explorer.Component("fileExplorer", files)
		if isHtml {
			if err := loadComponent.Render(c.Request.Context(), c.Writer); err != nil {
				c.Status(500)
				return
			}
		} else {
			c.JSON(200, gin.H{
				"message": "File explorer loaded successfully",
				"rootDir": rootDir,
			})
		}
	})
	apiRoute(apiV1Group, "POST", "/files/upload", func(c *gin.Context) {
		isHtml := c.GetHeader("Accept") == "text/html"
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
		newFilePath := filepath.Join(fileDir, header.Filename)
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
			loadComponent := file_explorer_load.Component("/")
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
	apiRoute(apiV1Group, "POST", "/update", func(c *gin.Context) {
		isHtml := c.GetHeader("Accept") == "text/html"
		var r update.UpdateRequest
		if err := c.BindJSON(&r); err != nil {
			if isHtml {
				c.Writer.WriteString(`<span class="text-red-500">Invalid request body</span>`)
			} else {
				c.JSON(400, gin.H{
					"error": "Invalid request body",
				})
			}
			return
		}
		if err := update.Update(r.Version); err != nil {
			if isHtml {
				c.Writer.WriteString(fmt.Sprintf(`<span class="text-red-500">%s</span>`, html.EscapeString(err.Error())))
			} else {
				c.JSON(500, gin.H{
					"error": err.Error(),
				})
			}
			return
		}
		go update.RestartAutobutler(2 * time.Second)
		if isHtml {
			c.Writer.WriteString(`<span class="text-green-500">Update successful, Autobutler will restart.</span>`)
		} else {
			c.JSON(200, gin.H{
				"message": "Update successful, Autobutler will restart.",
			})
		}
	})
}

func setupRoutes(router *gin.Engine) {
	setupStaticRoutes(router)
	setupUiRoutes(router)
	setupApiRoutes(router)
}

func apiRoute(router *gin.RouterGroup, method string, route string, handler func(c *gin.Context)) gin.IRoutes {
	route = util.TrimLeading(route, '/')
	switch method {
	case "GET":
		{
			return router.GET(route, handler)
		}
	case "POST":
		{
			return router.POST(route, handler)
		}
	case "PUT":
		{
			return router.PUT(route, handler)
		}
	case "DELETE":
		{
			return router.DELETE(route, handler)
		}
	default:
		{
			panic(fmt.Sprintf("Unsupported HTTP method: %s", method))
		}
	}
}

func uiRoute(router *gin.Engine, path string, handler func(c *gin.Context)) gin.IRoutes {
	path = util.TrimLeading(path, '/')
	route := fmt.Sprintf("/%s", path)
	return router.GET(route, handler)
}
