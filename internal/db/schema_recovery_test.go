package db

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// The migration set was regrouped by subject area in #1758, which renumbered it
// from 000-021 down to 001-007, and it has grown past 021 since. Every database
// built before that carries a version stamp from the old numbering, and
// initSchema has to notice: every one of those stamps but 0 is a number the set
// serves, and the database would otherwise be half-migrated in silence.
//
// Nothing here empties a database (#3127). One that cannot be carried forward
// is set aside whole when it is from the old numbering, and otherwise stops
// the server with the repair in the error.
//
// These tests cannot use the dbtest helper - it lives in a package that imports
// this one, and initSchema is unexported - so they open their databases the way
// migrations_test.go does.

// legacySchemaAtVersion21 is the shape a developer's quark.db had at the top of
// the old numbering: auth with is_admin (old 016), sessions with last_used_at
// (old 020), device_names still keyed by device_path, and an FTS5 table (old
// 019) whose shadow tables the drop has to cope with.
const legacySchemaAtVersion21 = `
CREATE TABLE users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	username TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	recovery_phrase_hash TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	is_admin INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE sessions (
	token TEXT PRIMARY KEY,
	user_id INTEGER NOT NULL,
	expires_at DATETIME NOT NULL,
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	last_used_at DATETIME NOT NULL DEFAULT (datetime('now')),
	FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
CREATE TABLE device_names (
	device_path TEXT PRIMARY KEY,
	display_name TEXT NOT NULL,
	updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE connected_devices (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	ip_address TEXT NOT NULL,
	user_agent TEXT NOT NULL DEFAULT '',
	UNIQUE (ip_address, user_agent)
);
CREATE VIRTUAL TABLE file_content_fts USING fts5(content);
`

// legacySchemaAtVersion3 is the same database eighteen migrations earlier, and
// the dangerous one: old 3 is a number the new set can serve. Old 002 had
// already created users, so the table is present - only is_admin,
// last_used_at and device_roles are missing.
const legacySchemaAtVersion3 = `
CREATE TABLE users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	username TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	recovery_phrase_hash TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE sessions (
	token TEXT PRIMARY KEY,
	user_id INTEGER NOT NULL,
	expires_at DATETIME NOT NULL,
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
CREATE TABLE device_names (
	device_path TEXT PRIMARY KEY,
	display_name TEXT NOT NULL,
	updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE connected_devices (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	ip_address TEXT NOT NULL,
	user_agent TEXT NOT NULL DEFAULT '',
	UNIQUE (ip_address, user_agent)
);
`

func openSchemaTestDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, _ := openSchemaTestDBAt(t)
	return conn
}

// openSchemaTestDBAt also returns the file's path, with symlinks resolved the
// way SQLite reports it, for the tests that look at the file itself.
func openSchemaTestDBAt(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("resolve temp dir: %v", err)
	}
	path := filepath.Join(dir, "quark.db")
	conn, err := sql.Open("sqlite", DSN(path))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn, path
}

// stampSchemaVersion writes the row golang-migrate would have written, in the
// table shape its sqlite driver creates.
func stampSchemaVersion(t *testing.T, conn *sql.DB, version int, dirty bool) {
	t.Helper()
	statements := []string{
		`CREATE TABLE IF NOT EXISTS schema_migrations (version uint64, dirty bool)`,
		`DELETE FROM schema_migrations`,
	}
	for _, statement := range statements {
		if _, err := conn.Exec(statement); err != nil {
			t.Fatalf("stamp %q: %v", statement, err)
		}
	}
	if _, err := conn.Exec(
		`INSERT INTO schema_migrations (version, dirty) VALUES (?, ?)`, version, dirty,
	); err != nil {
		t.Fatalf("stamp version: %v", err)
	}
}

func readSchemaVersion(t *testing.T, conn *sql.DB) (int, bool) {
	t.Helper()
	var version int
	var dirty bool
	if err := conn.QueryRow(`SELECT version, dirty FROM schema_migrations`).Scan(&version, &dirty); err != nil {
		t.Fatalf("read schema_migrations: %v", err)
	}
	return version, dirty
}

// latestSchemaVersion is read off a freshly migrated database rather than
// written down here, so regrouping the set again does not break these tests -
// which is the whole reason the regrouping needed recovery code in the first
// place.
func latestSchemaVersion(t *testing.T) int {
	t.Helper()
	conn := openSchemaTestDB(t)
	if err := initSchema(&DatabaseSqlc{Db: conn}); err != nil {
		t.Fatalf("migrate reference database: %v", err)
	}
	version, dirty := readSchemaVersion(t, conn)
	if dirty {
		t.Fatal("reference database came out dirty")
	}
	return version
}

func hasTable(t *testing.T, conn *sql.DB, table string) bool {
	t.Helper()
	var count int
	if err := conn.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE name = ?`, table,
	).Scan(&count); err != nil {
		t.Fatalf("look up %s: %v", table, err)
	}
	return count > 0
}

func hasColumn(t *testing.T, conn *sql.DB, table, column string) bool {
	t.Helper()
	var count int
	if err := conn.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, column,
	).Scan(&count); err != nil {
		t.Fatalf("inspect %s.%s: %v", table, column, err)
	}
	return count > 0
}

// assertNewSchema checks the markers that only the compressed set produces:
// device_names re-keyed by serial, the device_roles table that the old
// numbering did not have until 008, and the is_admin column it did not have
// until 016.
func assertNewSchema(t *testing.T, conn *sql.DB) {
	t.Helper()
	version, dirty := readSchemaVersion(t, conn)
	if want := latestSchemaVersion(t); version != want {
		t.Errorf("schema version = %d, want %d", version, want)
	}
	if dirty {
		t.Error("the database was left dirty")
	}
	if !hasColumn(t, conn, "device_names", "device_serial") {
		t.Error("device_names is not keyed by device_serial; the old schema survived")
	}
	if hasColumn(t, conn, "device_names", "device_path") {
		t.Error("device_names still has device_path; the old schema survived")
	}
	if !hasTable(t, conn, "device_roles") {
		t.Error("device_roles missing; 002_storage_devices did not run")
	}
	if !hasColumn(t, conn, "users", "is_admin") {
		t.Error("users.is_admin missing; 001_auth did not run")
	}
	if !hasTable(t, conn, "vfs_metadata") {
		t.Error("vfs_metadata missing; 007_vfs did not run")
	}
}

func applySchema(t *testing.T, conn *sql.DB, schema string) {
	t.Helper()
	if _, err := conn.Exec(schema); err != nil {
		t.Fatalf("apply fixture schema: %v", err)
	}
}

// seedUser adds the row the tests look for afterwards: in the original when
// Quark refused to start, in the file set aside when it started fresh.
func seedUser(t *testing.T, conn *sql.DB) {
	t.Helper()
	if _, err := conn.Exec(
		`INSERT INTO users (username, password_hash, recovery_phrase_hash) VALUES ('ada', 'x', 'y')`,
	); err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return contents
}

// setAsideFiles lists what initSchema renamed out of the way beside path.
func setAsideFiles(t *testing.T, path string) []string {
	t.Helper()
	matches, err := filepath.Glob(path + ".stale-*")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	return matches
}

// assertRefused holds a refusal to what an operator relies on: the server did
// not start, the message carries the facts and the way out, and the database
// is byte for byte what it was.
func assertRefused(t *testing.T, err error, path string, before []byte, want ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("initSchema started on a database it cannot carry forward")
	}
	want = append(want, path, fmt.Sprintf("mv '%s' '%s.stale'", path, path))
	for _, text := range want {
		if !strings.Contains(err.Error(), text) {
			t.Errorf("the refusal does not say %q:\n%v", text, err)
		}
	}
	if !bytes.Equal(readFile(t, path), before) {
		t.Error("the refused database was modified")
	}
	if aside := setAsideFiles(t, path); len(aside) != 0 {
		t.Errorf("the refused database was set aside: %v", aside)
	}
}

// A stamp above the top of the migration set is a downgrade: a newer build
// migrated this database and an older one is starting on it. The data is
// intact, so Quark refuses to start and touches nothing (#3127). The stamp is
// one past the top rather than a fixed number, because the set keeps growing
// and a fixed stamp eventually becomes a version the set can serve.
func TestInitSchemaRefusesAVersionAboveThisBuild(t *testing.T) {
	conn, path := openSchemaTestDBAt(t)
	if err := initSchema(&DatabaseSqlc{Db: conn}); err != nil {
		t.Fatalf("initial migrate: %v", err)
	}
	seedUser(t, conn)
	latest := latestSchemaVersion(t)
	stampSchemaVersion(t, conn, latest+1, false)
	before := readFile(t, path)

	err := initSchema(&DatabaseSqlc{Db: conn})

	assertRefused(t, err, path, before,
		fmt.Sprintf("schema version %d", latest+1),
		fmt.Sprintf("stop at %d", latest),
		"at or above",
	)
}

// A migration that died half-applied leaves nothing on disk saying which of
// its statements ran, but everything the earlier migrations built is still
// there. Quark refuses to start and prints the repair (#3127).
func TestInitSchemaRefusesADirtyDatabase(t *testing.T) {
	conn, path := openSchemaTestDBAt(t)
	if err := initSchema(&DatabaseSqlc{Db: conn}); err != nil {
		t.Fatalf("initial migrate: %v", err)
	}
	seedUser(t, conn)
	latest := latestSchemaVersion(t)
	stampSchemaVersion(t, conn, latest, true)
	before := readFile(t, path)

	err := initSchema(&DatabaseSqlc{Db: conn})

	assertRefused(t, err, path, before,
		fmt.Sprintf("schema version %d", latest),
		"dirty",
		fmt.Sprintf(`sqlite3 '%s' "UPDATE schema_migrations SET version = %d, dirty = 0"`, path, latest-1),
	)
}

// A stamp this build's set can serve, on a schema without the column that
// version's migrations create, and too high to be the old numbering: nothing
// explains it, so Quark does not guess (#3127).
func TestInitSchemaRefusesAVersionItCannotExplain(t *testing.T) {
	conn, path := openSchemaTestDBAt(t)
	applySchema(t, conn, legacySchemaAtVersion21)
	seedUser(t, conn)
	stampSchemaVersion(t, conn, legacyHighestVersion+1, false)
	before := readFile(t, path)

	err := initSchema(&DatabaseSqlc{Db: conn})

	assertRefused(t, err, path, before, fmt.Sprintf("schema version %d", legacyHighestVersion+1), "users.status")
}

// A database stamped under the pre-#1758 numbering cannot be carried forward
// by any migration, so it is renamed out of the way with its rows in it and
// Quark starts on a fresh one (#3127).
//
// Old 3 is the one the structural check was written for: the number alone
// says nothing is wrong. Old 16 to 21 are the ones it used to miss: is_admin
// arrived at old 016, the set has grown past 21, and such a database was
// migrated from 022 onto a schema with none of what 001 to 021 build. Old 0
// is below the set, which starts at 001.
func TestInitSchemaSetsAsideAnOldNumberingDatabase(t *testing.T) {
	cases := []struct {
		name    string
		schema  string
		version int
	}{
		{"old 0, below the set", legacySchemaAtVersion3, 0},
		{"old 3, without is_admin", legacySchemaAtVersion3, 3},
		{"old 13, without is_admin", legacySchemaAtVersion3, 13},
		{"old 16, with is_admin", legacySchemaAtVersion21, 16},
		{"old 21, with is_admin", legacySchemaAtVersion21, 21},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			conn, path := openSchemaTestDBAt(t)
			applySchema(t, conn, c.schema)
			seedUser(t, conn)
			stampSchemaVersion(t, conn, c.version, false)
			before := readFile(t, path)

			database := &DatabaseSqlc{Db: conn}
			if err := initSchema(database); err != nil {
				t.Fatalf("initSchema on an old-numbering database: %v", err)
			}
			t.Cleanup(func() { database.Db.Close() })

			// The handle the caller holds is the fresh database's.
			assertNewSchema(t, database.Db)
			var users int
			if err := database.Db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&users); err != nil {
				t.Fatalf("count users: %v", err)
			}
			if users != 0 {
				t.Errorf("the fresh database holds %d old users", users)
			}
			// A new install, with the id resetutil.Watch reads at startup.
			var installID string
			if err := database.Db.QueryRow(`SELECT install_id FROM install`).Scan(&installID); err != nil || installID == "" {
				t.Errorf("install id of the fresh database = %q, %v", installID, err)
			}

			aside := setAsideFiles(t, path)
			if len(aside) != 1 {
				t.Fatalf("set-aside files = %v, want exactly one", aside)
			}
			if !bytes.Equal(readFile(t, aside[0]), before) {
				t.Error("the set-aside file is not the old database, byte for byte")
			}
			old, err := sql.Open("sqlite", DSN(aside[0]))
			if err != nil {
				t.Fatalf("open the set-aside database: %v", err)
			}
			defer old.Close()
			var username string
			if err := old.QueryRow(`SELECT username FROM users`).Scan(&username); err != nil || username != "ada" {
				t.Errorf("the set-aside database lost its rows: %q, %v", username, err)
			}
			if version, _ := readSchemaVersion(t, old); version != c.version {
				t.Errorf("the set-aside database is stamped %d, want %d", version, c.version)
			}
		})
	}
}

// When the old database cannot be renamed there is no safe way to start
// fresh: Quark refuses, and never falls back to emptying it (#3127).
func TestInitSchemaRefusesWhenTheDatabaseCannotBeSetAside(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root renames inside a read-only directory")
	}
	conn, path := openSchemaTestDBAt(t)
	applySchema(t, conn, legacySchemaAtVersion3)
	seedUser(t, conn)
	stampSchemaVersion(t, conn, 3, false)
	before := readFile(t, path)

	// A directory nothing can be renamed in.
	dir := filepath.Dir(path)
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	err := initSchema(&DatabaseSqlc{Db: conn})

	assertRefused(t, err, path, before, "schema version 3", "permission denied")
}

// A first install has nothing recorded, and must migrate up rather than be
// treated as stale.
func TestInitSchemaMigratesFreshDatabaseWithoutRecovering(t *testing.T) {
	conn := openSchemaTestDB(t)
	migrationSource, err := newMigrationSource()
	if err != nil {
		t.Fatalf("open migration source: %v", err)
	}
	state, err := classifyMigrationState(conn, migrationSource)
	if err != nil {
		t.Fatalf("classifyMigrationState: %v", err)
	}
	if state.Kind != migrationStateCurrent {
		t.Fatalf("fresh database classified %d: %s", state.Kind, state.Reason)
	}

	if err := initSchema(&DatabaseSqlc{Db: conn}); err != nil {
		t.Fatalf("initSchema: %v", err)
	}
	assertNewSchema(t, conn)
}

// A database already correct under the new numbering must be left alone. The
// row is the point: it proves the recovery is a recovery and not a wipe on
// every boot.
func TestInitSchemaLeavesCurrentDatabaseUntouched(t *testing.T) {
	conn := openSchemaTestDB(t)
	if err := initSchema(&DatabaseSqlc{Db: conn}); err != nil {
		t.Fatalf("initial migrate: %v", err)
	}
	if _, err := conn.Exec(
		`INSERT INTO users (username, password_hash, recovery_phrase_hash) VALUES ('ada', 'x', 'y')`,
	); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	migrationSource, err := newMigrationSource()
	if err != nil {
		t.Fatalf("open migration source: %v", err)
	}
	state, err := classifyMigrationState(conn, migrationSource)
	if err != nil {
		t.Fatalf("classifyMigrationState: %v", err)
	}
	if state.Kind != migrationStateCurrent {
		t.Fatalf("current database classified %d: %s", state.Kind, state.Reason)
	}

	if err := initSchema(&DatabaseSqlc{Db: conn}); err != nil {
		t.Fatalf("second initSchema: %v", err)
	}

	var username string
	if err := conn.QueryRow(`SELECT username FROM users`).Scan(&username); err != nil {
		t.Fatalf("the existing account was destroyed by a boot that should have been a no-op: %v", err)
	}
	if username != "ada" {
		t.Errorf("username = %q, want ada", username)
	}
	assertNewSchema(t, conn)
}

// Whatever SQLite left beside the database belongs to it and goes with it: a
// journal left behind would be replayed into the fresh database.
func TestMoveAsideTakesTheJournalAndNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	path, aside := filepath.Join(dir, "quark.db"), filepath.Join(dir, "quark.db.stale-then")
	write := func(files map[string]string) {
		t.Helper()
		for file, contents := range files {
			if err := os.WriteFile(file, []byte(contents), 0o600); err != nil {
				t.Fatalf("write: %v", err)
			}
		}
	}
	write(map[string]string{path: "database", path + "-journal": "journal"})

	if err := moveAside(path, aside); err != nil {
		t.Fatalf("moveAside: %v", err)
	}
	if string(readFile(t, aside)) != "database" || string(readFile(t, aside+"-journal")) != "journal" {
		t.Error("the database and its journal were not both moved")
	}
	if _, err := os.Lstat(path + "-journal"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the journal was left beside the fresh database: %v", err)
	}

	// A second database must not land on the first one set aside.
	write(map[string]string{path: "second", path + "-journal": "second journal"})
	if err := moveAside(path, aside); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("moveAside onto a file already set aside = %v, want it refused", err)
	}
	if string(readFile(t, aside)) != "database" || string(readFile(t, path)) != "second" {
		t.Error("a refused moveAside moved the database")
	}

	// When it is the journal that cannot move, the database goes back where it
	// was: a database apart from its journal is not the database it was.
	if err := os.Remove(aside); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := moveAside(path, aside); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("moveAside onto a journal already set aside = %v, want it refused", err)
	}
	if string(readFile(t, path)) != "second" || string(readFile(t, aside+"-journal")) != "journal" {
		t.Error("a moveAside that failed part way did not put the database back")
	}
}

// Every version of the set, on the schema the set builds, carries forward.
// This is what holds the structural markers to the set: regroup it, or move
// the migration that creates one, and this fails before a device is set aside.
func TestClassifyMigrationStateCarriesEveryVersionForward(t *testing.T) {
	migrationSource, err := newMigrationSource()
	if err != nil {
		t.Fatalf("open migration source: %v", err)
	}
	for version := 1; version <= latestSchemaVersion(t); version++ {
		conn, _ := migrateTo(t, uint(version))
		state, err := classifyMigrationState(conn, migrationSource)
		if err != nil {
			t.Fatalf("classify version %d: %v", version, err)
		}
		if state.Kind != migrationStateCurrent {
			t.Errorf("version %d classified %d: %s", version, state.Kind, state.Reason)
		}
	}
}
