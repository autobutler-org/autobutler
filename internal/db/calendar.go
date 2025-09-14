package db

import (
	"database/sql"
	"fmt"
)

const DefaultCalendarId = 1

type Calendar struct {
	ID   int
	Name string
}

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

func (d *Database) QueryCalendar(id int) (*Calendar, error) {
	if d == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	query := "SELECT * FROM calendars WHERE id = ?"
	rows, err := d.Db.Query(query, id)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %s", query)
	}
	defer rows.Close()
	calendars, err := NewCalendarFromRows(rows)
	if err != nil {
		return nil, fmt.Errorf("error creating calendars from rows: %w", err)
	}
	if len(calendars) == 0 {
		return nil, nil
	}
	return calendars[0], nil
}

func (d *Database) UpsertCalendar(newCalendar Calendar) (*Calendar, error) {
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
		calendar, err := d.QueryCalendar(newCalendar.ID)
		if calendar != nil {
			// Calendar exists, update it
			_, err = d.Exec(
				`UPDATE calendars SET
					name = ?
						WHERE id = ?`,
				newCalendar.Name,
				newCalendar.ID,
			)
			if err != nil {
				return nil, fmt.Errorf("error updating calendar: %w", err)
			}
		} else {
			// Calendar does not exist, insert it
			result, err := d.Exec(
				`INSERT INTO calendars (
					name
				) VALUES (
				 	?
				)`,
				newCalendar.Name,
			)
			if err != nil {
				return nil, fmt.Errorf("error inserting calendar: %w", err)
			}
			newId, err := result.LastInsertId()
			if err != nil {
				return nil, fmt.Errorf("error getting last insert id: %w", err)
			}
			newCalendar.ID = int(newId)
		}
	}
	calendar, err := d.QueryCalendar(newCalendar.ID)
	if err != nil {
		return nil, fmt.Errorf("error querying calendar after insert/update: %w", err)
	}
	return calendar, nil
}
