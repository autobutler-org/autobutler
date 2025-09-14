package db

import (
	"autobutler/pkg/util"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

//go:embed ddl
var ddl embed.FS

var (
	Instance Database
)

type Database struct {
	Db *sql.DB
}

func (d *Database) Exec(query string, args ...any) (sql.Result, error) {
	if d == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return d.Db.Exec(query, args...)
}

func (d *Database) Query(query string, args ...any) (*sql.Rows, error) {
	if d == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return d.Query(query, args...)
}

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
	files, err := ddl.ReadDir("ddl")
	if err != nil {
		return fmt.Errorf("failed to read ddl directory: %w", err)
	}
	for _, file := range files {
		schemaFilename := filepath.Join("ddl", file.Name())
		schemaBytes, err := ddl.ReadFile(schemaFilename)
		if err != nil {
			return fmt.Errorf("failed to read schema file(%s): %w", schemaFilename, err)
		}

		schema := string(schemaBytes)
		_, err = Instance.Exec(schema)
		if err != nil {
			return fmt.Errorf("failed to execute schema(%s): %w", schemaFilename, err)
		}
	}
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
