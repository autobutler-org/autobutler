package db

import (
	"autobutler/pkg/calendar"
	"autobutler/pkg/util"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var DatabaseQueries *Queries

func init() {
	dataDir := util.GetDataDir()
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		panic(fmt.Sprintf("failed to create data directory: %v", err))
	}

	dataFilePath := filepath.Join(dataDir, "autobutler.db")

	Instance.Db, err = sql.Open("sqlite", dataFilePath)
	if err != nil {
		panic(fmt.Sprintf("failed to open database: %v", err))
	}
	sqlConn, err := Instance.Db.Conn(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database connection: %v", err))
	}
	DatabaseQueries = New(sqlConn)

	if err := initSchema(); err != nil {
		panic(fmt.Sprintf("failed to initialize database schema: %v", err))
	}
	if err := seedData(); err != nil {
		panic(fmt.Sprintf("failed to seed database: %v", err))
	}
}

func seedData() error {
	var err error
	calendar := NewCalendar("Defaults")
	// Make sure that we only create a first calendar, and not continue to make more
	calendar.ID = DefaultCalendarId
	calendar, err = Instance.UpsertCalendar(*calendar)
	if err != nil || calendar == nil {
		return fmt.Errorf("failed to insert/update default calendar: %w", err)
	}
	err = seedTestCalendarEvents()
	if err != nil {
		return fmt.Errorf("failed to seed test calendar events: %w", err)
	}
	return nil
}

func seedTestCalendarEvents() error {
	now := time.Now()
	events := []*calendar.CalendarEvent{
		calendar.NewCalendarEventWithEnd(
			"Meeting with Bingus",
			"Discuss project updates",
			time.Date(now.Year(), now.Month(), 10, 14, 0, 0, 0, time.UTC),
			time.Date(now.Year(), now.Month(), 10, 14, 1, 0, 0, time.UTC),
			false,
			"Somewhere",
			DefaultCalendarId,
		),
		calendar.NewCalendarEventWithEnd(
			"Meeting with Bingus's dumb cat",
			"Discuss project updates",
			time.Date(now.Year(), now.Month(), 10, 14, 1, 0, 0, time.UTC),
			time.Date(now.Year(), now.Month(), 10, 15, 2, 0, 0, time.UTC),
			false,
			"Somewhere else",
			DefaultCalendarId,
		),
		calendar.NewCalendarEvent(
			"Conference",
			"Annual tech conference",
			time.Date(now.Year(), now.Month(), 20, 0, 0, 0, 0, time.UTC),
			true,
			"Convention Center",
			DefaultCalendarId,
		),
	}
	for i, event := range events {
		event.ID = int64(i + 1)
		_, err := Instance.UpsertCalendarEvent(*event)
		if err != nil {
			return fmt.Errorf("failed to insert test calendar event: %w", err)
		}
	}
	return nil
}
