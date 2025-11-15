package ui

import (
	"autobutler/pkg/calendar"
	"autobutler/pkg/ui/types"
	"autobutler/pkg/ui/views"
	"autobutler/pkg/util/ctxutil"
	"autobutler/pkg/util/deputil"
	"autobutler/pkg/util/serverutil"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

func SetupCalendarRoutes(router *gin.Engine) {
	serverutil.UiRoute(router, "/calendar", func(c *gin.Context) templ.Component {
		deps, _ := ctxutil.Get[deputil.Dependencies](c, "deps")
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
		return views.CalendarWithTime(deps, types.NewPageState(), targetTime)
	})
}
