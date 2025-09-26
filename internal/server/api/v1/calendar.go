package v1

import (
	cal "autobutler/internal/server/ui/components/calendar"
	"autobutler/internal/server/ui/components/calendar/event_editor"
	"autobutler/pkg/calendar"
	"autobutler/pkg/db"
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func SetupCalendarRoutes(apiV1Group *gin.RouterGroup) {
	deleteCalendarEvent(apiV1Group)
	getCalendarEvent(apiV1Group)
	newCalendarEvent(apiV1Group)
	updateCalendarEvent(apiV1Group)
}

func deleteCalendarEvent(apiV1Group *gin.RouterGroup) {
	apiRoute(apiV1Group, "DELETE", "/calendar/events/:eventId", func(c *gin.Context) {
		eventId, err := strconv.Atoi(c.Param("eventId"))
		if err != nil {
			c.Writer.WriteString(`<span class="text-red-500">Invalid event ID</span>`)
			c.Status(400)
			return
		}
		if err := db.Instance.DeleteCalendarEvent(eventId); err != nil {
			c.Writer.WriteString(`<span class="text-red-500">` + err.Error() + `</span>`)
			c.Status(500)
			return
		}
		if err := cal.Component(calendar.CalendarViewMonth).Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
	})
}

func getCalendarEvent(apiV1Group *gin.RouterGroup) {
	apiRoute(apiV1Group, "GET", "/calendar/:eventId", func(c *gin.Context) {
		eventId, err := strconv.Atoi(c.Param("eventId"))
		if err != nil {
			c.Writer.WriteString(`<span class="text-red-500">Invalid event ID</span>`)
			c.Status(400)
			return
		}

		event, err := db.DatabaseQueries.GetCalendarEvent(context.Background(), int64(eventId))
		if err != nil {
			c.Writer.WriteString(`<span class="text-red-500">` + err.Error() + `</span>`)
			c.Status(404)
			return
		}
		if err := event_editor.ComponentWithEvent(*db.NewCalendarEvent(event)).Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(500)
			return
		}
		c.Status(200)
	})
}

func newCalendarEvent(apiV1Group *gin.RouterGroup) {
	apiRoute(apiV1Group, "POST", "/calendar/events", func(c *gin.Context) {
		yearString := c.PostForm("year")
		monthString := c.PostForm("month")
		dayString := c.PostForm("day")
		title := c.PostForm("title")
		startTimeString := c.PostForm("startTime")
		endTimeString := c.PostForm("endTime")
		description := c.PostForm("description")
		location := c.PostForm("location")
		startTime, err := makeTime(yearString, monthString, dayString, startTimeString)
		if err != nil {
			c.Writer.WriteString(`<span class="text-red-500">Invalid start time: ` + err.Error() + `</span>`)
			c.Status(400)
			return
		}
		var calendarEvent *calendar.CalendarEvent
		if endTimeString == "" {
			calendarEvent = calendar.NewCalendarEvent(
				title,
				description,
				*startTime,
				false,
				location,
				db.DefaultCalendarId,
			)
		} else {
			endTime, err := makeTime(yearString, monthString, dayString, endTimeString)
			if err != nil {
				c.Writer.WriteString(`<span class="text-red-500">Invalid end time: ` + err.Error() + `</span>`)
				c.Status(400)
				return
			}
			calendarEvent = calendar.NewCalendarEventWithEnd(
				title,
				description,
				*startTime,
				*endTime,
				false,
				location,
				db.DefaultCalendarId,
			)
		}
		if _, err := db.Instance.UpsertCalendarEvent(*calendarEvent); err != nil {
			c.Writer.WriteString(`<span class="text-red-500">` + err.Error() + `</span>`)
			c.Status(500)
			return
		}
		if err := cal.Component(calendar.CalendarViewMonth).Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
	})
}

func updateCalendarEvent(apiV1Group *gin.RouterGroup) {
	apiRoute(apiV1Group, "PUT", "/calendar/events", func(c *gin.Context) {
		eventId := c.PostForm("id")
		yearString := c.PostForm("year")
		monthString := c.PostForm("month")
		dayString := c.PostForm("day")
		title := c.PostForm("title")
		startTimeString := c.PostForm("startTime")
		endTimeString := c.PostForm("endTime")
		description := c.PostForm("description")
		location := c.PostForm("location")
		startTime, err := makeTime(yearString, monthString, dayString, startTimeString)
		if err != nil {
			c.Writer.WriteString(`<span class="text-red-500">Invalid start time: ` + err.Error() + `</span>`)
			c.Status(400)
			return
		}
		var calendarEvent *calendar.CalendarEvent
		if endTimeString == "" {
			calendarEvent = calendar.NewCalendarEvent(
				title,
				description,
				*startTime,
				false,
				location,
				db.DefaultCalendarId,
			)
		} else {
			endTime, err := makeTime(yearString, monthString, dayString, endTimeString)
			if err != nil {
				c.Writer.WriteString(`<span class="text-red-500">Invalid end time: ` + err.Error() + `</span>`)
				c.Status(400)
				return
			}
			calendarEvent = calendar.NewCalendarEventWithEnd(
				title,
				description,
				*startTime,
				*endTime,
				false,
				location,
				db.DefaultCalendarId,
			)
		}
		if eventId != "" {
			eventId, err := strconv.Atoi(eventId)
			calendarEvent.ID = int64(eventId)
			if err != nil {
				c.Writer.WriteString(`<span class="text-red-500">Invalid event ID</span>`)
				c.Status(400)
				return
			}
		}
		if _, err := db.Instance.UpsertCalendarEvent(*calendarEvent); err != nil {
			c.Writer.WriteString(`<span class="text-red-500">` + err.Error() + `</span>`)
			c.Status(500)
			return
		}
		if err := cal.Component(calendar.CalendarViewMonth).Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
	})
}

func makeTime(yearString, monthString, dayString string, startTime string) (*time.Time, error) {
	year, err := strconv.Atoi(yearString)
	if err != nil {
		return nil, err
	}
	month, err := strconv.Atoi(monthString)
	if err != nil {
		return nil, err
	}
	day, err := strconv.Atoi(dayString)
	if err != nil {
		return nil, err
	}
	hour, err := strconv.Atoi(startTime[0:2])
	if err != nil {
		return nil, err
	}
	minute, err := strconv.Atoi(startTime[3:5])
	if err != nil {
		return nil, err
	}
	t := time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.UTC)
	return &t, nil
}
