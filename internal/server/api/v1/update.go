package v1

import (
	"autobutler/internal/update"
	"fmt"
	"html"
	"time"

	"github.com/gin-gonic/gin"
)

func SetupUpdateRoutes(apiV1Group *gin.RouterGroup) {
	updateRoute(apiV1Group)
}

func updateRoute(apiV1Group *gin.RouterGroup) {
	apiRoute(apiV1Group, "POST", "/update", func(c *gin.Context) {
		version := c.PostForm("version")
		if err := update.Update(version); err != nil {
			fmt.Fprintf(c.Writer, `<span class="text-red-500">%s</span>`, html.EscapeString(err.Error()))
			return
		}
		go update.RestartAutobutler(2 * time.Second)
		c.Writer.WriteString(`<span class="text-green-500">Update successful, Autobutler will restart.</span>`)
	})
}
