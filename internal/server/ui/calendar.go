package ui

import (
	"autobutler/internal/server/ui/types"
	"autobutler/internal/server/ui/views"
	"autobutler/internal/serverutil"
	"autobutler/pkg/calendar"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

func SetupCalendarRoutes(router *gin.Engine) {
	serverutil.UiRoute(router, "/calendar", func(c *gin.Context) templ.Component {
		yearStr := c.Query("year")
		monthStr := c.Query("month")

		var targetTime *time.Time
		if yearStr != "" && monthStr != "" {
			year, err := strconv.Atoi(yearStr)
			if err == nil {
				// Try parsing as month name first, then fall back to number
				month := calendar.ParseMonth(monthStr)
				if month.IsValid() {
					t := time.Date(year, month.ToTimeMonth(), 1, 0, 0, 0, 0, time.UTC)
					targetTime = &t
				}
			}
		}
		return views.CalendarWithTime(types.NewPageState(), targetTime)
	})
}
