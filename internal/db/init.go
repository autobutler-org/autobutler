package db

import (
	"autobutler/pkg/util/fileutil"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

var DatabaseQueries *Queries

func init() {
	dataDir := fileutil.GetDataDir()
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		panic(fmt.Sprintf("failed to create data directory: %v", err))
	}

	dataFilePath := filepath.Join(dataDir, "autobutler.db")
	healthFilePath := filepath.Join(dataDir, "autobutler.health.db")

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

	// Initialize health database for OTEL traces (no migrations needed)
	HealthInstance.Db, err = sql.Open("sqlite", healthFilePath)
	if err != nil {
		panic(fmt.Sprintf("failed to open health database: %v", err))
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
	return nil
}
