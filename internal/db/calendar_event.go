package db

import (
	"autobutler/pkg/calendar"
	"database/sql"
	"fmt"
	"time"
)

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

func (d *Database) QueryCalendarEvent(id int) (*calendar.CalendarEvent, error) {
	if d == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	query := "SELECT * FROM calendar_events WHERE id = ?"
	rows, err := d.Db.Query(query, id)
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
	return calendarEvents[0], nil
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

func (d *Database) UpsertCalendarEvent(newCalendarEvent calendar.CalendarEvent) (*calendar.CalendarEvent, error) {
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
		calendarEvent, err := d.QueryCalendarEvent(newCalendarEvent.ID)
		if calendarEvent != nil {
			// Calendar event exists, update it
			_, err = d.Exec(
				`UPDATE calendar_events SET
					title = ?,
					description = ?,
					start_time = ?,
					end_time = ?,
					all_day = ?,
					location = ?,
					calendar_id = ?
						WHERE id = ?`,
				newCalendarEvent.Title,
				newCalendarEvent.Description,
				newCalendarEvent.StartTime,
				newCalendarEvent.EndTime,
				newCalendarEvent.AllDay,
				newCalendarEvent.Location,
				newCalendarEvent.CalendarID,
				newCalendarEvent.ID,
			)
			if err != nil {
				return nil, fmt.Errorf("error updating calendar event: %w", err)
			}
		} else {
			// Calendar event does not exist, insert it
			result, err := d.Exec(
				`INSERT INTO calendar_events (
					title,
					description,
					start_time,
					end_time,
					all_day,
					location,
					calendar_id
				) VALUES (
				 	?,
					?,
					?,
					?,
					?,
					?,
					?
				)`,
				newCalendarEvent.Title,
				newCalendarEvent.Description,
				newCalendarEvent.StartTime,
				newCalendarEvent.EndTime,
				newCalendarEvent.AllDay,
				newCalendarEvent.Location,
				newCalendarEvent.CalendarID,
			)
			if err != nil {
				return nil, fmt.Errorf("error inserting calendar event: %w", err)
			}
			newId, err := result.LastInsertId()
			if err != nil {
				return nil, fmt.Errorf("error getting last insert id: %w", err)
			}
			newCalendarEvent.ID = int(newId)
		}
	}
	calendarEvent, err := d.QueryCalendarEvent(newCalendarEvent.ID)
	if err != nil {
		return nil, fmt.Errorf("error querying calendar event after insert/update: %w", err)
	}
	return calendarEvent, nil
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
		event.ID = i + 1
		_, err := Instance.UpsertCalendarEvent(*event)
		if err != nil {
			return fmt.Errorf("failed to insert test calendar event: %w", err)
		}
	}
	return nil
}
