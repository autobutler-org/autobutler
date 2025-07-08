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
