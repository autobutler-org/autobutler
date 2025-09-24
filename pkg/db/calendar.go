package db

import (
	"context"
	"database/sql"
	"fmt"
)

const DefaultCalendarId = 1

func NewCalendar(name string) *Calendar {
	return &Calendar{
		Name: name,
	}
}

func NewCalendarFromRows(rows *sql.Rows) ([]*Calendar, error) {
	var calendars []*Calendar
	for rows.Next() {
		var calendar Calendar
		if err := rows.Scan(&calendar.ID, &calendar.Name); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		calendars = append(calendars, &calendar)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}
	return calendars, nil
}

func (d *Database) UpsertCalendar(newCalendar Calendar) (*Calendar, error) {
	if d == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	calendar, err := DatabaseQueries.GetCalendar(context.Background(), newCalendar.ID)
	if err == nil {
		// Calendar exists, update it
		calendar, err = DatabaseQueries.UpdateCalendar(
			context.Background(),
			UpdateCalendarParams{
				ID:   newCalendar.ID,
				Name: newCalendar.Name,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("error updating calendar: %w", err)
		}
	} else {
		// Calendar does not exist, insert it
		calendar, err = DatabaseQueries.CreateCalendar(
			context.Background(),
			newCalendar.Name,
		)
		if err != nil {
			return nil, fmt.Errorf("error inserting calendar: %w", err)
		}
		newCalendar.ID = calendar.ID
	}
	return &calendar, nil
}
