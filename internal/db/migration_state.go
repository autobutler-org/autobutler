package db

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4/source"
)

// migrationStateKind is which case classifyMigrationState found.
type migrationStateKind int

const (
	// migrationStateCurrent is a database the embedded migrations carry
	// forward: nothing recorded yet, or a version of this set on the schema
	// this set builds. The migrations run up.
	migrationStateCurrent migrationStateKind = iota
	// migrationStateDirty is a migration left half-applied.
	migrationStateDirty
	// migrationStateNewer is a version above this build's highest: a
	// downgrade.
	migrationStateNewer
	// migrationStateLegacyNumbering is a version stamped under the numbering
	// the set had before #1758.
	migrationStateLegacyNumbering
	// migrationStateUnknown is everything else. Nothing Quark knows of
	// produces it.
	migrationStateUnknown
)

// migrationState is what classifyMigrationState read, and what it made of it.
type migrationState struct {
	Kind migrationStateKind
	// Version and Dirty are the row in schema_migrations, zero when there is
	// none.
	Version uint
	Dirty   bool
	// First and Highest bound this build's migration set.
	First, Highest uint
	// Reason says, in words, why Kind is not migrationStateCurrent.
	Reason string
}

const (
	// authMigrationVersion is 001_auth, which creates users.is_admin.
	authMigrationVersion = 1
	// userStatusMigrationVersion is 011_user_status, which adds users.status.
	userStatusMigrationVersion = 11
	// legacyHighestVersion is the highest version the numbering before #1758
	// ever stamped.
	legacyHighestVersion = 21
)

// classifyMigrationState reads the migration state recorded in sqlDB and says
// whether the embedded migration set can carry it forward. It only reads, and
// it decides nothing about what happens next: initSchema acts on the answer,
// and so can any other backend's startup.
//
// Four things make a recorded state unusable:
//
//   - The dirty flag is set. A migration was interrupted or failed, and
//     nothing on disk says which. Checked first: whatever else is true of the
//     database, that has to be repaired before anything runs on it.
//
//   - The version is above this build's highest. A newer build migrated the
//     database and an older one is starting on it. golang-migrate refuses
//     outright: "no migration found for version 33".
//
//   - The version was stamped under the old numbering. Regrouping the set by
//     subject area (#1758) renumbered 000-021 down to 001-007, and the set
//     has since grown past 021, so every old stamp but 0 is a number this
//     set serves. This one is silent. Old 3 meant connected_devices, auth and
//     device names; new 003 means auth, storage devices and connected
//     devices. golang-migrate compares numbers and inspects nothing, so it
//     would apply 004 onward onto a schema the earlier migrations never
//     built.
//
//   - Anything else: a version inside the set's range that the set does not
//     hold, or a schema that matches neither numbering.
//
// A column on users is what separates the old numbering from a legitimately
// older database, and which column depends on the version, because the old
// numbering gained users.is_admin part way up:
//
//   - Below 011, users.is_admin. Every version of this set is at or above
//     001_auth, which creates it; the old numbering did not until 016. A
//     users table alone proves nothing, since old 002 created one.
//
//   - From 011 up, users.status, which 011_user_status adds and the old
//     numbering never had at any version. is_admin cannot serve here: an old
//     database at 16, 19, 20 or 21 has it.
//
// A version this set serves, without its column, is from the old numbering
// when it is one the old numbering could have stamped, and unexplained above
// that. TestClassifyMigrationStateCarriesEveryVersionForward holds both
// columns to the set: regrouping it again, or moving either migration, fails
// that test.
func classifyMigrationState(sqlDB *sql.DB, migrationSource source.Driver) (migrationState, error) {
	first, highest, err := migrationSetBounds(migrationSource)
	if err != nil {
		return migrationState{}, err
	}
	state := migrationState{First: first, Highest: highest}

	var tableExists int
	if err := sqlDB.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'schema_migrations'`,
	).Scan(&tableExists); err != nil {
		return state, fmt.Errorf("failed to look for schema_migrations: %w", err)
	}
	if tableExists == 0 {
		return state, nil
	}

	err = sqlDB.QueryRow(`SELECT version, dirty FROM schema_migrations LIMIT 1`).Scan(&state.Version, &state.Dirty)
	if errors.Is(err, sql.ErrNoRows) {
		// The table is created before the first migration runs, so an empty one
		// is a fresh database.
		return state, nil
	}
	if err != nil {
		return state, fmt.Errorf("failed to read the recorded schema version: %w", err)
	}
	version := state.Version

	if state.Dirty {
		state.Kind = migrationStateDirty
		state.Reason = fmt.Sprintf("migration %d was interrupted or failed, and the database is marked dirty", version)
		return state, nil
	}
	if version > highest {
		state.Kind = migrationStateNewer
		state.Reason = "a newer Quark build migrated this database"
		return state, nil
	}
	if version < first {
		// Only the old numbering had a migration below this set's first.
		state.Kind = migrationStateLegacyNumbering
		state.Reason = fmt.Sprintf("version %d is below this build's first migration, and only the numbering "+
			"used before the migrations were regrouped had one", version)
		return state, nil
	}

	body, _, err := migrationSource.ReadUp(version)
	if errors.Is(err, fs.ErrNotExist) {
		state.Kind = migrationStateUnknown
		state.Reason = fmt.Sprintf("version %d is inside the range of this build's migrations, but is not one of them", version)
		return state, nil
	}
	if err != nil {
		return state, fmt.Errorf("failed to look up migration %d: %w", version, err)
	}
	if err := body.Close(); err != nil {
		return state, fmt.Errorf("failed to close migration %d: %w", version, err)
	}

	column, createdBy := "is_admin", authMigrationVersion
	if version >= userStatusMigrationVersion {
		column, createdBy = "status", userStatusMigrationVersion
	}
	var hasColumn int
	if err := sqlDB.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info('users') WHERE name = ?`, column,
	).Scan(&hasColumn); err != nil {
		return state, fmt.Errorf("failed to inspect the users table: %w", err)
	}
	if hasColumn > 0 {
		return state, nil
	}
	missing := fmt.Sprintf("users.%s, which migration %03d creates, is missing", column, createdBy)
	if version > legacyHighestVersion {
		state.Kind = migrationStateUnknown
		state.Reason = fmt.Sprintf("version %d is one of this build's migrations, but %s", version, missing)
		return state, nil
	}
	state.Kind = migrationStateLegacyNumbering
	state.Reason = fmt.Sprintf("version %d was stamped under the numbering used before the migrations were "+
		"regrouped: %s", version, missing)
	return state, nil
}

// migrationSetBounds returns the lowest and highest version in the set.
func migrationSetBounds(migrationSource source.Driver) (first, highest uint, err error) {
	first, err = migrationSource.First()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to find the first migration: %w", err)
	}
	highest = first
	for {
		next, err := migrationSource.Next(highest)
		if errors.Is(err, fs.ErrNotExist) {
			return first, highest, nil
		}
		if err != nil {
			return 0, 0, fmt.Errorf("failed to find the migration after %d: %w", highest, err)
		}
		highest = next
	}
}

// databaseFilePath asks SQLite which file sqlDB has open. It is the path the
// refusal names and the one set aside, so it is the file in use and not a
// guess at it.
func databaseFilePath(sqlDB *sql.DB) (string, error) {
	var path string
	if err := sqlDB.QueryRow(`SELECT file FROM pragma_database_list WHERE name = 'main'`).Scan(&path); err != nil {
		return "", fmt.Errorf("failed to ask the database for its file: %w", err)
	}
	return path, nil
}

// setAsideDatabase renames the database file out of the way, with its rows
// in it, and leaves database holding a fresh, empty file at the same path. It
// returns where the old one went.
//
// The handle is closed first. A rename under an open handle moves nothing as
// far as that handle is concerned: its descriptors follow the inode, so the
// migrations would run into the file just set aside. Nothing else holds the
// handle yet, because this runs before ConnectToDatabase has returned it.
//
// Nothing is copied into the fresh database. The tables a reset keeps
// (keptTables) arrived in 031, and a database from the old numbering predates
// it: what they hold is still in the data directory's settings.json and
// account-request-history.jsonl, which the server imports on this start as it
// would have after a reset. The migrations issue the fresh database its own
// install id, before resetutil.Watch first reads one.
//
// ponytail: two processes starting on one old-numbering file at the same
// moment can each set a file aside, the second one the first's fresh database.
// Nothing is lost, but they end up on different files; a lock file fixes that
// if SQLite ever serves more than one instance.
func setAsideDatabase(database *DatabaseSqlc, path string) (string, error) {
	if path == "" {
		return "", errors.New("the database is not a file")
	}
	if err := database.Db.Close(); err != nil {
		return "", fmt.Errorf("failed to close the database: %w", err)
	}
	aside := path + ".stale-" + time.Now().UTC().Format("20060102T150405Z")
	if err := moveAside(path, aside); err != nil {
		return "", err
	}
	fresh, err := sql.Open("sqlite", DSN(path))
	if err != nil {
		return "", fmt.Errorf("failed to open a fresh database: %w", err)
	}
	database.Db = fresh
	database.Queries = pooledQueries(fresh)
	return aside, nil
}

// moveAside renames the database at path to aside, and with it whatever
// SQLite left beside it: a journal that stayed behind would be replayed into
// the fresh database. It overwrites nothing, and when a rename fails it puts
// back what it had already moved, so the database is either set aside whole
// or still where it was.
func moveAside(path, aside string) error {
	moved := make([]string, 0, 4)
	for _, suffix := range []string{"", "-journal", "-wal", "-shm"} {
		err := renameWithoutOverwriting(path+suffix, aside+suffix)
		if suffix != "" && errors.Is(err, fs.ErrNotExist) {
			// SQLite left no such file, which is the usual case.
			continue
		}
		if err != nil {
			for _, done := range moved {
				err = errors.Join(err, os.Rename(aside+done, path+done))
			}
			return err
		}
		moved = append(moved, suffix)
	}
	return nil
}

// renameWithoutOverwriting is os.Rename, which replaces a file already at to,
// made to refuse instead.
func renameWithoutOverwriting(from, to string) error {
	if _, err := os.Lstat(to); err == nil {
		return fmt.Errorf("rename %s %s: %w", from, to, fs.ErrExist)
	}
	return os.Rename(from, to)
}

// refuseToStart is the error that stops the server on a database it will
// neither migrate nor set aside. Nothing can be served, so the error is the
// whole interface: it says what was found, why Quark left it alone, and what
// to do next, down to the command.
func refuseToStart(state migrationState, path string, migrationSource source.Driver, setAsideErr error) error {
	dirty := "not dirty"
	if state.Dirty {
		dirty = "dirty"
	}
	var steps string
	switch {
	case state.Kind == migrationStateLegacyNumbering:
		steps = fmt.Sprintf("No migration carries this database forward, so Quark sets it aside and starts on a\n"+
			"fresh one. It could not:\n\n    %v\n\n"+
			"Check that the user Quark runs as may rename files in that directory, then start\n"+
			"Quark again, or rename the database yourself with the mv command in this message.", setAsideErr)
	case state.Kind == migrationStateNewer:
		steps = fmt.Sprintf("Install the Quark release that last ran on this machine, or any later one: a build\n"+
			"whose migrations are at or above %d. It starts on this database as it is.", state.Version)
	case state.Kind == migrationStateDirty && state.Version >= state.First && state.Version <= state.Highest:
		steps = dirtyRepairSteps(state, path, migrationSource)
	case state.Kind == migrationStateDirty:
		steps = fmt.Sprintf("This build has no migration %d. Install the Quark release that last ran on this\n"+
			"machine and start it: it prints the steps that repair the database.", state.Version)
	default:
		steps = "Nothing Quark knows of leaves a database in this state, so there is no repair to\n" +
			"offer. If a different Quark build last ran on this machine, install that one."
	}
	return fmt.Errorf("this build of Quark will not start on this database, and has not changed it.\n\n"+
		"  Database: %s\n"+
		"  Found:    schema version %d, %s. This build's migrations stop at %d.\n"+
		"  Why:      %s.\n"+
		"            Quark never empties or guesses at a database it cannot carry forward:\n"+
		"            the accounts, albums and vault entries in it are still there.\n\n"+
		"What to do:\n\n%s\n\n"+
		"To start over instead, set the database aside and start Quark again. Quark creates a\n"+
		"fresh database and comes up at first-run setup, and the old one stays in the renamed\n"+
		"file. Stored files are not touched.\n\n"+
		"    mv %s %s",
		path, state.Version, dirty, state.Highest, state.Reason, steps,
		shellQuote(path), shellQuote(path+".stale"))
}

// dirtyRepairSteps is the manual repair for a dirty database, which leans on
// one fact: golang-migrate's sqlite driver runs a migration in a single
// transaction, so the migration was applied whole or not at all, and applying
// it again either works or changes nothing.
func dirtyRepairSteps(state migrationState, path string, migrationSource source.Driver) string {
	quoted := shellQuote(path)
	again := `DELETE FROM schema_migrations`
	if previous, err := migrationSource.Prev(state.Version); err == nil {
		again = fmt.Sprintf(`UPDATE schema_migrations SET version = %d, dirty = 0`, previous)
	}
	return fmt.Sprintf("With Quark stopped, and the sqlite3 command-line tool:\n\n"+
		"1. Keep a copy.\n\n"+
		"    cp %[1]s %[2]s\n\n"+
		"2. Have Quark apply migration %[3]d again, and start it.\n\n"+
		"    sqlite3 %[1]s \"%[4]s\"\n\n"+
		"   A migration runs in one transaction: it applies in full or changes nothing.\n\n"+
		"3. If that start fails with \"failed to run migrations\", read the SQL error beside it.\n"+
		"   \"already exists\" or \"duplicate column name\" means migration %[3]d had been applied\n"+
		"   before it was interrupted. Record that, and start Quark.\n\n"+
		"    sqlite3 %[1]s \"UPDATE schema_migrations SET version = %[3]d, dirty = 0\"\n\n"+
		"   Any other error means migration %[3]d cannot run on this data. Put the copy from\n"+
		"   step 1 back and report the error: https://github.com/autobutler-org/quark/issues",
		quoted, shellQuote(path+".before-repair"), state.Version, again)
}

// shellQuote quotes path for a POSIX shell, for a command an operator pastes.
func shellQuote(path string) string {
	return "'" + strings.ReplaceAll(path, "'", `'\''`) + "'"
}
