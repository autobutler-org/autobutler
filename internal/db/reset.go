package db

import (
	"database/sql"
	"fmt"
)

// ResetDatabase drops every object the application owns and re-runs the
// migrations from scratch, leaving a database at first-boot state on the same
// connection the caller already holds.
//
// It deliberately does not unlink quark.db. The process opens that file once at
// startup and keeps a *sql.DB and a *sql.Conn for its whole lifetime, and
// removing a file out from under a live handle disconnects nothing: the inode
// survives until the last descriptor closes, so queries keep succeeding against
// a file that no longer has a name and the reset only appears to have happened
// after a restart. Dropping and re-migrating in place keeps every handle valid,
// re-runs the migrations from 000, and needs no restart.
//
// golang-migrate's own Drop() cannot stand in for this: it issues DROP TABLE
// for every row in sqlite_master, sqlite_sequence included, and SQLite refuses
// to drop that one.
func ResetDatabase(database *DatabaseSqlc) error {
	if database == nil || database.Db == nil {
		return fmt.Errorf("database not initialized")
	}
	if err := dropAllObjects(database.Db); err != nil {
		return fmt.Errorf("failed to drop database objects: %w", err)
	}
	if err := initSchema(database); err != nil {
		return fmt.Errorf("failed to re-run migrations: %w", err)
	}
	return nil
}

// dropAllObjects removes every table and view outside SQLite's own sqlite_%
// namespace. Triggers and indexes go with the tables that own them, and
// dropping an FTS5 virtual table takes its shadow tables with it — which is why
// each statement is IF EXISTS: the shadow rows are still in the snapshot of
// sqlite_master this read from.
func dropAllObjects(sqlDB *sql.DB) error {
	const listObjects = `
		SELECT type, name FROM sqlite_master
		WHERE type IN ('table', 'view') AND name NOT LIKE 'sqlite_%'`

	rows, err := sqlDB.Query(listObjects)
	if err != nil {
		return fmt.Errorf("failed to list database objects: %w", err)
	}

	type object struct{ kind, name string }
	objects := make([]object, 0)
	for rows.Next() {
		var o object
		if err := rows.Scan(&o.kind, &o.name); err != nil {
			rows.Close()
			return fmt.Errorf("failed to read database object: %w", err)
		}
		objects = append(objects, o)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("failed to list database objects: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("failed to close object listing: %w", err)
	}

	for _, o := range objects {
		// The kind is one of the two literals in listObjects and the name comes
		// from sqlite_master, so neither is caller-controlled; DDL takes no
		// bound parameters for identifiers in any case.
		statement := fmt.Sprintf(`DROP %s IF EXISTS "%s"`, o.kind, o.name)
		if _, err := sqlDB.Exec(statement); err != nil {
			return fmt.Errorf("failed to drop %s %s: %w", o.kind, o.name, err)
		}
	}
	return nil
}
