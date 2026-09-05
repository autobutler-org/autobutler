package db

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// The embedded migration set must apply cleanly to an empty database and leave
// the schema the rest of the codebase queries. It deliberately asserts on
// tables and columns rather than on a version number: the set is regrouped by
// subject area rather than appended to, so the version is an implementation
// detail and pinning it only breaks the next regrouping (#1758).
func TestMigrationsApplyCleanly(t *testing.T) {
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

	tables := []string{
		"users", "sessions",
		"device_names", "device_roles",
		"connected_devices",
		"photo_albums", "photo_album_items", "photo_rotations", "photo_favorites", "photo_hashes",
		"vault_config", "vault_folders", "vault_entries", "vault_location",
		"file_content", "file_content_fts",
		"vfs_metadata", "vfs_db_entries",
	}
	for _, table := range tables {
		var count int
		if err := conn.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE name = ?`, table,
		).Scan(&count); err != nil {
			t.Fatalf("look up %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("table %s missing after migration", table)
		}
	}

	columns := map[string][]string{
		"users":        {"is_admin"},
		"sessions":     {"last_used_at"},
		"photo_hashes": {"dhash", "content_hash"},
	}
	for table, names := range columns {
		for _, name := range names {
			var count int
			if err := conn.QueryRow(
				`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, name,
			).Scan(&count); err != nil {
				t.Fatalf("inspect %s: %v", table, err)
			}
			if count != 1 {
				t.Errorf("%s.%s missing after migration", table, name)
			}
		}
	}

	// The vault location singleton is seeded by the migration, not by the app.
	var located int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM vault_location WHERE id = 1`).Scan(&located); err != nil {
		t.Fatalf("count vault_location: %v", err)
	}
	if located != 1 {
		t.Errorf("vault_location seed row missing, got %d rows", located)
	}
}
