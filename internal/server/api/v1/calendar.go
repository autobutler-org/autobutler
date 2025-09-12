package v1

import (
	"autobutler/internal/server/ui/components/calendar/event_viewer"
	"autobutler/pkg/calendar"

	"github.com/gin-gonic/gin"
)

func SetupCalendarRoutes(apiV1Group *gin.RouterGroup) {
	getCalendarEvent(apiV1Group)
}

func getCalendarEvent(apiV1Group *gin.RouterGroup) {
	apiRoute(apiV1Group, "GET", "/calendar/:eventId", func(c *gin.Context) {
		eventId := c.Param("eventId")
		event, err := calendar.GetEventByID(eventId)
		if err != nil {
			c.Writer.WriteString(`<span class="text-red-500">` + err.Error() + `</span>`)
			c.Status(404)
			return
		}
		eventViewer := event_viewer.Component(event)
		if err := eventViewer.Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(500)
			return
		}
		c.Status(200)
	})
}
