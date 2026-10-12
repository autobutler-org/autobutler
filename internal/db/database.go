// Package db is Quark's SQLite layer: the connection, the golang-migrate migrations under migrations/, and the
// query methods sqlc generates from sql/queries/. Code outside this package reaches the database through those
// generated methods.
package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"

	// Registers the "sqlite" database/sql driver used by every connection here.
	_ "modernc.org/sqlite"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

type DatabaseSqlc struct {
	Db      *sql.DB
	Queries *Queries
}

// InTx runs fn against queries bound to one transaction on sqlDB, committing
// only if fn returns nil. An error from fn rolls the transaction back and comes
// back as it is, so callers can still match it with errors.Is; a panic in fn
// rolls back too and keeps unwinding.
func InTx(ctx context.Context, sqlDB *sql.DB, fn func(*Queries) error) error {
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	// A no-op once Commit has run.
	defer tx.Rollback()

	if err := fn(New(tx)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
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

//go:embed migrations
var migrations embed.FS

// newMigrationSource opens the embedded migration set.
func newMigrationSource() (source.Driver, error) {
	migrationSource, err := iofs.New(migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to create iofs source: %w", err)
	}
	return migrationSource, nil
}

// initSchema brings the database up to this build's schema, and never empties
// one to get there (#3127). What it does with a database whose recorded
// version the migrations cannot carry forward depends on which case
// classifyMigrationState found:
//
//   - Stamped under the numbering from before #1758: no build can migrate it,
//     so the file is renamed aside with its rows in it and the migrations run
//     on a fresh one. database then holds the new handle, and the Quark comes
//     up at first-run setup.
//
//   - Dirty, newer than this build, or anything unexplained: the data is
//     probably intact, so nothing is touched and the error, which stops the
//     server, carries the repair.
func initSchema(database *DatabaseSqlc) error {
	migrationSource, err := newMigrationSource()
	if err != nil {
		return err
	}

	// Checked before the migrate instance is built, because building one
	// creates schema_migrations and an empty table is indistinguishable from a
	// database that never had one.
	state, err := classifyMigrationState(database.Db, migrationSource)
	if err != nil {
		return err
	}
	if state.Kind != migrationStateCurrent {
		path, err := databaseFilePath(database.Db)
		if err != nil {
			return err
		}
		if state.Kind != migrationStateLegacyNumbering {
			return refuseToStart(state, path, migrationSource, nil)
		}
		aside, err := setAsideDatabase(database, path)
		if err != nil {
			return refuseToStart(state, path, migrationSource, err)
		}
		slog.Warn(
			"set this machine's database aside and started on a fresh one: the schema "+
				"version recorded in it is from the migration numbering used before #1758, "+
				"which no migration can carry forward. Nothing was deleted: the old database "+
				"is in the renamed file, and the Quark comes back at first-run setup.",
			"reason", state.Reason, "database", path, "setAside", aside,
		)
	}

	driver, err := sqlite.WithInstance(database.Db, &sqlite.Config{})
	if err != nil {
		return fmt.Errorf("failed to create sqlite driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", migrationSource, "sqlite", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	// ErrNoChange just means the schema is already at the latest migration.
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}
