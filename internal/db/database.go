package db

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	// Registers the "sqlite" database/sql driver used by every connection here.
	_ "modernc.org/sqlite"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

type DatabaseSqlc struct {
	Db      *sql.DB
	Queries *Queries
}

type DatabaseRaw struct {
	Db *sql.DB
}

func (d *DatabaseSqlc) Exec(query string, args ...any) (sql.Result, error) {
	if d == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return d.Db.Exec(query, args...)
}

func (d *DatabaseSqlc) Query(query string, args ...any) (*sql.Rows, error) {
	if d == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return d.Db.Query(query, args...)
}

func (d *DatabaseRaw) Exec(query string, args ...any) (sql.Result, error) {
	if d == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return d.Db.Exec(query, args...)
}

func (d *DatabaseRaw) Query(query string, args ...any) (*sql.Rows, error) {
	if d == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return d.Db.Query(query, args...)
}

//go:embed migrations
var migrations embed.FS

func initSchema(database *DatabaseSqlc) error {
	driver, err := sqlite.WithInstance(database.Db, &sqlite.Config{})
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
	// ErrNoChange just means the schema is already at the latest migration.
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}
