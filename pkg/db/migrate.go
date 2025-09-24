package db

import (
	"autobutler/pkg/util"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations
var migrations embed.FS

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

	if err := initSchema(); err != nil {
		panic(fmt.Sprintf("failed to initialize database schema: %v", err))
	}
	if err := seedData(); err != nil {
		panic(fmt.Sprintf("failed to seed database: %v", err))
	}
}

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
