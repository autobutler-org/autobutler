package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/autobutler-org/quark/pkg/util/storageutil"
)

// ConnectToDatabase opens the database the server runs on: the one
// ConfigFromEnv names, which is SQLite's quark.db in the data directory unless
// the environment asks for PostgreSQL.
func ConnectToDatabase() (*DatabaseSqlc, error) {
	config, err := ConfigFromEnv()
	if err != nil {
		return nil, err
	}
	if config.Backend == Postgres {
		return ConnectToPostgres(config.PostgresURL)
	}

	var database DatabaseSqlc
	dataDir := storageutil.GetDataDir()
	err = os.MkdirAll(dataDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	dataFilePath := filepath.Join(dataDir, "quark.db")

	database.Db, err = sql.Open("sqlite", DSN(dataFilePath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}
	database.Queries = pooledQueries(database.Db)

	if err := initSchema(&database); err != nil {
		return nil, fmt.Errorf("failed to initialize database schema: %v", err)
	}

	return &database, nil
}

// maxOpenConns bounds the pool every sqlc query draws from. Each connection
// costs a file descriptor and its own page cache, and the device has four
// cores, so more connections than this only queue inside SQLite instead of
// inside database/sql.
const maxOpenConns = 16

// pooledQueries binds Queries to sqlDB's connection pool, bounded to
// maxOpenConns, so one request's query never waits for another's (#2766).
//
// A caller's cancellation reaches its query. The driver answers a canceled
// context with sqlite3_interrupt, which interrupts the connection rather than
// the one query (#2743); a pooled connection belongs to one caller until its
// statement or rows are done, so the interrupt fails nobody else, and a
// request whose client went away stops costing anything.
//
// The bound covers everything else that uses sqlDB too. A caller that holds
// one connection and asks for a second (a query run while iterating another's
// rows, or outside a transaction from inside it) waits for a free one, so
// keep such nesting off hot paths.
func pooledQueries(sqlDB *sql.DB) Querier {
	sqlDB.SetMaxOpenConns(maxOpenConns)
	// Idle connections are kept, not closed: the default of two would reopen a
	// connection, and rerun its pragmas, on most requests under load.
	sqlDB.SetMaxIdleConns(maxOpenConns)
	return NewQueries(sqlDB)
}
