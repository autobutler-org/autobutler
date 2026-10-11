package db

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/stdlib"

	"github.com/autobutler-org/quark/internal/db/pgdb"
)

// Backend names a database engine Quark can run on.
type Backend string

const (
	// SQLite is the default: one file in the data directory.
	SQLite Backend = "sqlite"
	// Postgres is a PostgreSQL server reached over the network (#2955).
	Postgres Backend = "postgres"
)

const (
	// BackendEnv is the environment variable that picks the backend. Unset or
	// empty means SQLite.
	BackendEnv = "QUARK_DB_BACKEND"
	// DatabaseURLEnv is the environment variable holding the PostgreSQL
	// connection URL. Only the PostgreSQL backend reads it.
	DatabaseURLEnv = "QUARK_DATABASE_URL"
)

// Config is which database the server opens. ConfigFromEnv is the one place
// that reads it from the environment.
type Config struct {
	Backend Backend
	// PostgresURL is the connection URL, set only when Backend is Postgres.
	// It carries the password, so it is never logged or put in an error.
	PostgresURL string
}

// ConfigFromEnv reads the backend choice from BackendEnv and DatabaseURLEnv.
// An unknown backend, or PostgreSQL without a URL, is an error that says what
// to set: a server that guessed would open the wrong database.
func ConfigFromEnv() (Config, error) {
	switch backend := Backend(strings.TrimSpace(os.Getenv(BackendEnv))); backend {
	case "", SQLite:
		return Config{Backend: SQLite}, nil
	case Postgres:
		url := strings.TrimSpace(os.Getenv(DatabaseURLEnv))
		if url == "" {
			return Config{}, fmt.Errorf(
				"%s=%s needs %s: set it to the database's URL, for example postgres://user:password@host:5432/quark",
				BackendEnv, Postgres, DatabaseURLEnv)
		}
		return Config{Backend: Postgres, PostgresURL: url}, nil
	default:
		return Config{}, fmt.Errorf(
			"%s=%q is not a database backend: set it to %q (the default when unset) or %q",
			BackendEnv, string(backend), SQLite, Postgres)
	}
}

// BackendOf reports which engine sqlDB is connected to, from the driver it
// was opened with. The pool is the one thing every caller already holds, so
// reading the engine off it means there is no second value to keep in step.
func BackendOf(sqlDB *sql.DB) Backend {
	if _, ok := sqlDB.Driver().(*stdlib.Driver); ok {
		return Postgres
	}
	return SQLite
}

// Backend reports which engine the database is on.
func (d *DatabaseSqlc) Backend() Backend {
	return BackendOf(d.Db)
}

// UnsupportedError reports a feature that does not run on the PostgreSQL
// backend yet. Such a feature is built on SQL only SQLite accepts, outside
// what sqlc generates, and RequireSQLite turns it away before any of that SQL
// is sent: PostgreSQL would answer it with a syntax error, or a missing table,
// that tells nobody what is wrong.
type UnsupportedError struct {
	// Feature names what was asked for, worded to start a sentence.
	Feature string
	// Issue is the GitHub issue that tracks making it work.
	Issue int
}

func (e *UnsupportedError) Error() string {
	return fmt.Sprintf("%s is not supported on the PostgreSQL backend yet (#%d)", e.Feature, e.Issue)
}

// HTTPStatus is the status serverutil answers with when a handler returns
// this error: the request was understood, and this server cannot do it.
func (e *UnsupportedError) HTTPStatus() int {
	return http.StatusNotImplemented
}

// RequireSQLite returns an *UnsupportedError naming feature and its tracking
// issue when sqlDB is not a SQLite database, and nil when it is. It is the
// guard in front of every statement that only SQLite accepts; nothing else
// outside this package should need to know the engine.
func RequireSQLite(sqlDB *sql.DB, feature string, issue int) error {
	if BackendOf(sqlDB) == SQLite {
		return nil
	}
	return &UnsupportedError{Feature: feature, Issue: issue}
}

// queriesFor returns backend's generated queries bound to conn: a pool, one
// connection or a transaction.
func queriesFor(backend Backend, conn DBTX) Querier {
	if backend == Postgres {
		return &postgresQueries{q: pgdb.New(conn)}
	}
	return New(conn)
}
