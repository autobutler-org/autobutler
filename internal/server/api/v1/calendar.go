package v1

import (
	"autobutler/internal/db"
	cal "autobutler/internal/server/ui/components/calendar"
	"autobutler/internal/server/ui/components/calendar/event_editor"
	"autobutler/pkg/api"
	"autobutler/pkg/calendar"
	"autobutler/pkg/util/serverutil"
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func SetupCalendarRoutes(apiV1Group *gin.RouterGroup) {
	deleteCalendarEvent(apiV1Group)
	getCalendarEvent(apiV1Group)
	getCalendarMonth(apiV1Group)
	newCalendarEvent(apiV1Group)
	updateCalendarEvent(apiV1Group)
}

func deleteCalendarEvent(apiV1Group *gin.RouterGroup) {
	serverutil.ApiRoute(apiV1Group, "DELETE", "/calendar/events/:eventId", func(c *gin.Context) *api.Response {
		eventId, err := strconv.Atoi(c.Param("eventId"))
		if err != nil {
			return api.NewResponse().WithStatusCode(400).WithData(`<span class="text-red-500">Invalid event ID</span>`)
		}

		viewYearString := c.Query("viewYear")
		viewMonthString := c.Query("viewMonth")

		if err := db.Instance.DeleteCalendarEvent(eventId); err != nil {
			return api.NewResponse().WithStatusCode(500).WithData(`<span class="text-red-500">` + err.Error() + `</span>`)
		}

		// Return to the month the user was viewing
		if viewYearString != "" && viewMonthString != "" {
			viewYear, err := strconv.Atoi(viewYearString)
			if err == nil {
				viewMonth, err := strconv.Atoi(viewMonthString)
				if err == nil && viewMonth >= 1 && viewMonth <= 12 {
					targetTime := time.Date(viewYear, time.Month(viewMonth), 1, 0, 0, 0, 0, time.UTC)
					if err := cal.ComponentWithTime(calendar.CalendarViewMonth, targetTime).Render(c.Request.Context(), c.Writer); err != nil {
						return api.NewResponse().WithStatusCode(400)
					}
					return api.Ok()
				}
			}
		}

		// Fallback to current month if no view context provided
		if err := cal.Component(calendar.CalendarViewMonth).Render(c.Request.Context(), c.Writer); err != nil {
			return api.NewResponse().WithStatusCode(400)
		}
		return api.Ok()
	})
}

func getCalendarEvent(apiV1Group *gin.RouterGroup) {
	serverutil.ApiRoute(apiV1Group, "GET", "/calendar/:eventId", func(c *gin.Context) *api.Response {
		eventId, err := strconv.Atoi(c.Param("eventId"))
		if err != nil {
			return api.NewResponse().WithStatusCode(400).WithData(`<span class="text-red-500">Invalid event ID</span>`)
		}

		event, err := db.DatabaseQueries.GetCalendarEvent(context.Background(), int64(eventId))
		if err != nil {
			return api.NewResponse().WithStatusCode(404).WithData(`<span class="text-red-500">Event not found</span>`)
		}
		if err := event_editor.ComponentWithEvent(*db.NewCalendarEvent(event)).Render(c.Request.Context(), c.Writer); err != nil {
			return api.NewResponse().WithStatusCode(500)
		}
		return api.Ok()
	})
}

func getCalendarMonth(apiV1Group *gin.RouterGroup) {
	serverutil.ApiRoute(apiV1Group, "GET", "/calendar/month", func(c *gin.Context) *api.Response {
		yearStr := c.Query("year")
		monthStr := c.Query("month")

		year, err := strconv.Atoi(yearStr)
		if err != nil {
			return api.NewResponse().WithStatusCode(400).WithData(`<span class="text-red-500">Invalid year</span>`)
		}

		month, err := strconv.Atoi(monthStr)
		if err != nil {
			return api.NewResponse().WithStatusCode(400).WithData(`<span class="text-red-500">Invalid month</span>`)
		}

		targetTime := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		if err := cal.ComponentWithTime(calendar.CalendarViewMonth, targetTime).Render(c.Request.Context(), c.Writer); err != nil {
			return api.NewResponse().WithStatusCode(500)
		}
		return api.Ok()
	})
}

func newCalendarEvent(apiV1Group *gin.RouterGroup) {
	serverutil.ApiRoute(apiV1Group, "POST", "/calendar/events", func(c *gin.Context) *api.Response {
		yearString := c.PostForm("year")
		monthString := c.PostForm("month")
		dayString := c.PostForm("day")
		title := c.PostForm("title")
		startTimeString := c.PostForm("startTime")
		endTimeString := c.PostForm("endTime")
		description := c.PostForm("description")
		location := c.PostForm("location")
		viewYearString := c.PostForm("viewYear")
		viewMonthString := c.PostForm("viewMonth")

		startTime, err := makeTime(yearString, monthString, dayString, startTimeString)
		if err != nil {
			return api.NewResponse().WithStatusCode(400).WithData(`<span class="text-red-500">Invalid start time: ` + err.Error() + `</span>`)
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
				return api.NewResponse().WithStatusCode(400).WithData(`<span class="text-red-500">Invalid end time: ` + err.Error() + `</span>`)
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
			return api.NewResponse().WithStatusCode(500).WithData(`<span class="text-red-500">` + err.Error() + `</span>`)
		}

		// Return to the month the user was viewing
		if viewYearString != "" && viewMonthString != "" {
			viewYear, err := strconv.Atoi(viewYearString)
			if err == nil {
				viewMonth, err := strconv.Atoi(viewMonthString)
				if err == nil && viewMonth >= 1 && viewMonth <= 12 {
					targetTime := time.Date(viewYear, time.Month(viewMonth), 1, 0, 0, 0, 0, time.UTC)
					if err := cal.ComponentWithTime(calendar.CalendarViewMonth, targetTime).Render(c.Request.Context(), c.Writer); err != nil {
						return api.NewResponse().WithStatusCode(400)
					}
					return api.Ok()
				}
			}
		}

		// Fallback to current month if no view context provided
		if err := cal.Component(calendar.CalendarViewMonth).Render(c.Request.Context(), c.Writer); err != nil {
			return api.NewResponse().WithStatusCode(400)
		}
		return api.Ok()
	})
}

func updateCalendarEvent(apiV1Group *gin.RouterGroup) {
	serverutil.ApiRoute(apiV1Group, "PUT", "/calendar/events", func(c *gin.Context) *api.Response {
		eventId := c.PostForm("id")
		yearString := c.PostForm("year")
		monthString := c.PostForm("month")
		dayString := c.PostForm("day")
		title := c.PostForm("title")
		startTimeString := c.PostForm("startTime")
		endTimeString := c.PostForm("endTime")
		description := c.PostForm("description")
		location := c.PostForm("location")
		viewYearString := c.PostForm("viewYear")
		viewMonthString := c.PostForm("viewMonth")

		startTime, err := makeTime(yearString, monthString, dayString, startTimeString)
		if err != nil {
			return api.NewResponse().WithStatusCode(400).WithData(`<span class="text-red-500">Invalid start time: ` + err.Error() + `</span>`)
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
				return api.NewResponse().WithStatusCode(400).WithData(`<span class="text-red-500">` + "Invalid end time: " + err.Error() + `</span>`)
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
				return api.NewResponse().WithStatusCode(400).WithData(`<span class="text-red-500">` + "Invalid event ID: " + err.Error() + `</span>`)
			}
		}
		if _, err := db.Instance.UpsertCalendarEvent(*calendarEvent); err != nil {
			return api.NewResponse().WithStatusCode(500).WithData(`<span class="text-red-500">` + err.Error() + `</span>`)
		}

		// Return to the month the user was viewing
		if viewYearString != "" && viewMonthString != "" {
			viewYear, err := strconv.Atoi(viewYearString)
			if err == nil {
				viewMonth, err := strconv.Atoi(viewMonthString)
				if err == nil && viewMonth >= 1 && viewMonth <= 12 {
					targetTime := time.Date(viewYear, time.Month(viewMonth), 1, 0, 0, 0, 0, time.UTC)
					if err := cal.ComponentWithTime(calendar.CalendarViewMonth, targetTime).Render(c.Request.Context(), c.Writer); err != nil {
						return api.NewResponse().WithStatusCode(400)
					}
					return api.Ok()
				}
			}
		}

		// Fallback to current month if no view context provided
		if err := cal.Component(calendar.CalendarViewMonth).Render(c.Request.Context(), c.Writer); err != nil {
			return api.NewResponse().WithStatusCode(400)
		}
		return api.Ok()
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
