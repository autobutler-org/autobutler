package db

import (
	"database/sql"
	"embed"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

var (
	Instance       Database
	HealthInstance Database
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
