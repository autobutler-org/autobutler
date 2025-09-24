package db

import (
	"autobutler/pkg/calendar"
	"context"
	"database/sql"
	"fmt"
	"time"
)

func NewCalendarEvent(
	event CalendarEvent,
) *calendar.CalendarEvent {
	var endTime *time.Time = nil
	if event.EndTime.Valid {
		endTime = &event.EndTime.Time
	}
	return &calendar.CalendarEvent{
		ID:          event.ID,
		Title:       event.Title,
		Description: event.Description.String,
		StartTime:   event.StartTime,
		EndTime:     endTime,
		AllDay:      event.AllDay,
		Location:    event.Location.String,
		CalendarID:  event.CalendarID,
	}
}

func NewCalendarEventFromRows(rows *sql.Rows) ([]*calendar.CalendarEvent, error) {
	var calendarEvents []*calendar.CalendarEvent
	for rows.Next() {
		var calendarEvent calendar.CalendarEvent
		if err := rows.Scan(
			&calendarEvent.ID,
			&calendarEvent.Title,
			&calendarEvent.Description,
			&calendarEvent.StartTime,
			&calendarEvent.EndTime,
			&calendarEvent.AllDay,
			&calendarEvent.Location,
			&calendarEvent.CalendarID,
		); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		calendarEvents = append(calendarEvents, &calendarEvent)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}
	return calendarEvents, nil
}

func (d *Database) DeleteCalendarEvent(id int) error {
	if d == nil {
		return fmt.Errorf("database not initialized")
	}
	_, err := d.Exec("DELETE FROM calendar_events WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("error deleting calendar event: %w", err)
	}
	return nil
}

func (d *Database) QueryCalendarEventsForMonth(calendarId int, year int, month time.Month, includeEnds bool) (calendar.EventMap, error) {
	if d == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	dt := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	startTime := time.Date(dt.Year(), month, 1, 0, 0, 0, 0, time.UTC)
	endTime := startTime.AddDate(0, 1, 0)
	if includeEnds {
		monthInfo := calendar.NewMonthInfoFromTime(dt)
		startTime = startTime.AddDate(0, 0, -monthInfo.LeadingDays)
		endTime = startTime.AddDate(0, 0, monthInfo.TotalDays-monthInfo.LeadingDays)
	}
	query := "SELECT * FROM calendar_events WHERE calendar_id = ? AND start_time >= ? AND start_time <= ?"
	rows, err := d.Db.Query(
		query,
		calendarId,
		startTime,
		endTime,
	)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %s", query)
	}
	defer rows.Close()
	calendarEvents, err := NewCalendarEventFromRows(rows)
	if err != nil {
		return nil, fmt.Errorf("error creating calendar_events from rows: %w", err)
	}
	if len(calendarEvents) == 0 {
		return nil, nil
	}
	eventMap := calendar.EventMap{}
	for _, event := range calendarEvents {
		day := event.StartTime.Day()
		if _, exists := eventMap[day]; !exists {
			eventMap[day] = []*calendar.CalendarEvent{event}
			continue
		}
		eventMap[day] = append(eventMap[day], event)
	}
	return eventMap, nil
}

func (d *Database) UpsertCalendarEvent(newCalendarEvent calendar.CalendarEvent) (*CalendarEvent, error) {
	if d == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	// Start a transaction
	{
		tx, err := d.Db.Begin()
		if err != nil {
			return nil, fmt.Errorf("error starting transaction: %w", err)
		}
		// Defer rollback or commit of transaction
		defer func() {
			if err != nil {
				if rollbackErr := tx.Rollback(); rollbackErr != nil {
					fmt.Printf("error rolling back transaction: %v\n", rollbackErr)
				}
			} else {
				if commitErr := tx.Commit(); commitErr != nil {
					fmt.Printf("error committing transaction: %v\n", commitErr)
				}
			}
		}()
		calendarEvent, err := DatabaseQueries.GetCalendarEvent(context.Background(), newCalendarEvent.ID)
		if err == nil {
			// Calendar event exists, update it
			endTime := sql.NullTime{}
			if newCalendarEvent.EndTime != nil {
				endTime.Time = *newCalendarEvent.EndTime
				endTime.Valid = true
			}
			calendarEvent, err = DatabaseQueries.UpdateCalendarEvent(
				context.Background(),
				UpdateCalendarEventParams{
					Title: newCalendarEvent.Title,
					Description: sql.NullString{
						String: newCalendarEvent.Description,
						Valid:  true,
					},
					StartTime: newCalendarEvent.StartTime,
					EndTime:   endTime,
					AllDay:    newCalendarEvent.AllDay,
					Location: sql.NullString{
						String: newCalendarEvent.Location,
						Valid:  newCalendarEvent.Location != "",
					},
					CalendarID: newCalendarEvent.CalendarID,
				},
			)
			if err != nil {
				return nil, fmt.Errorf("error updating calendar event: %w", err)
			}
		} else {
			// Calendar event does not exist, insert it
			endTime := sql.NullTime{}
			if newCalendarEvent.EndTime != nil {
				endTime.Time = *newCalendarEvent.EndTime
				endTime.Valid = true
			}
			calendarEvent, err = DatabaseQueries.CreateCalendarEvent(
				context.Background(),
				CreateCalendarEventParams{
					Title: newCalendarEvent.Title,
					Description: sql.NullString{
						String: newCalendarEvent.Description,
						Valid:  true,
					},
					StartTime: newCalendarEvent.StartTime,
					EndTime:   endTime,
					AllDay:    newCalendarEvent.AllDay,
					Location: sql.NullString{
						String: newCalendarEvent.Location,
						Valid:  newCalendarEvent.Location != "",
					},
					CalendarID: newCalendarEvent.CalendarID,
				},
			)
			if err != nil {
				return nil, fmt.Errorf("error inserting calendar event: %w", err)
			}
			newCalendarEvent.ID = calendarEvent.ID
		}
		return &calendarEvent, nil
	}
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
