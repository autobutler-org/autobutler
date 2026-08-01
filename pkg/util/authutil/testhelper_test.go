package authutil_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/util/authutil"
	_ "modernc.org/sqlite"
)

// newTOTPTestDB creates an in-memory SQLite database with the full auth schema.
// Used across admin, TOTP, pairing, and share link tests.
func newTOTPTestDB(t *testing.T) *db.Queries {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	schema := `
		CREATE TABLE IF NOT EXISTS users (
			id                   INTEGER PRIMARY KEY AUTOINCREMENT,
			username             TEXT NOT NULL UNIQUE,
			password_hash        TEXT NOT NULL,
			recovery_phrase_hash TEXT NOT NULL,
			created_at           DATETIME NOT NULL DEFAULT (datetime('now')),
			is_admin             INTEGER NOT NULL DEFAULT 0,
			totp_secret          TEXT,
			totp_pending         TEXT
		);
		CREATE TABLE IF NOT EXISTS sessions (
			token      TEXT PRIMARY KEY,
			user_id    INTEGER NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
		);
		CREATE TABLE IF NOT EXISTS share_links (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			token_hash    TEXT    NOT NULL UNIQUE,
			created_by    INTEGER REFERENCES users(id) ON DELETE SET NULL,
			resource_type TEXT    NOT NULL CHECK(resource_type IN ('file', 'folder')),
			resource_path TEXT    NOT NULL,
			device_serial TEXT    NOT NULL DEFAULT '',
			expires_at    DATETIME,
			view_count    INTEGER NOT NULL DEFAULT 0,
			created_at    DATETIME NOT NULL DEFAULT (datetime('now'))
		);
	`

	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		t.Fatalf("get conn: %v", err)
	}
	if _, err := conn.ExecContext(context.Background(), schema); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	return db.New(conn)
}

// createTOTPTestUser inserts a test user (non-admin) and returns their ID.
func createTOTPTestUser(t *testing.T, queries *db.Queries) int64 {
	t.Helper()
	hash, err := authutil.HashPassword("testpassword")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user, err := queries.CreateUser(context.Background(), db.CreateUserParams{
		Username:           "testuser",
		PasswordHash:       hash,
		RecoveryPhraseHash: hash,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user.ID
}
