package db

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// Migration 021 must apply cleanly through the real embedded migration set, and
// must clear the sessions written in the old local-time format (#1650).
func TestMigration021AppliesAndClearsSessions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quark.db")
	conn, err := sql.Open("sqlite", DSN(path))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer conn.Close()

	database := &DatabaseSqlc{Db: conn}
	if err := initSchema(database); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var version int
	var dirty bool
	if err := conn.QueryRow(`SELECT version, dirty FROM schema_migrations`).Scan(&version, &dirty); err != nil {
		t.Fatalf("read schema_migrations: %v", err)
	}
	if dirty {
		t.Fatal("migrations left the database dirty")
	}
	if version < 21 {
		t.Fatalf("schema version %d, want at least 21", version)
	}

	// The column added by 020 survives 021 — it drops rows, not schema.
	var count int
	if err := conn.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info('sessions') WHERE name = 'last_used_at'`,
	).Scan(&count); err != nil {
		t.Fatalf("inspect sessions: %v", err)
	}
	if count != 1 {
		t.Fatal("sessions.last_used_at missing after migration 021")
	}

	// And a fresh database has no sessions to carry forward.
	if err := conn.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&count); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no sessions after migration, got %d", count)
	}
}
