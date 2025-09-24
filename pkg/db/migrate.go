package db

import (
	"embed"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations
var migrations embed.FS

func initSchema() error {
	driver, err := sqlite.WithInstance(Instance.Db, &sqlite.Config{})
	if err != nil {
		return fmt.Errorf("failed to create sqlite driver: %w", err)
	}
	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create iofs source: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	m.Up()
	return nil
}

func seedData() error {
	var err error
	calendar := NewCalendar("Default")
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
