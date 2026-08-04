package server

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/autobutler-org/autobutler/internal/db"
	_ "modernc.org/sqlite"
)

// testSchema is the minimal schema needed to exercise DeleteExpiredSessions.
const purgerTestSchema = `
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    recovery_phrase_hash TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS sessions (
    token TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
`

type purgerTestDB struct {
	q   *db.Queries
	raw *sql.DB
}

func newPurgerTestDB(t *testing.T) purgerTestDB {
	t.Helper()
	rawDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := rawDB.Exec(purgerTestSchema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	// Insert the required user row (sessions FK → users).
	if _, err := rawDB.Exec(
		`INSERT INTO users (username, password_hash, recovery_phrase_hash) VALUES ('testuser', 'hash', 'rhash')`,
	); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { rawDB.Close() })
	return purgerTestDB{q: db.New(rawDB), raw: rawDB}
}

func insertSession(t *testing.T, p purgerTestDB, token string, expiresAt time.Time) {
	t.Helper()
	if _, err := p.q.CreateSession(context.Background(), db.CreateSessionParams{
		Token:     token,
		UserID:    1,
		ExpiresAt: expiresAt,
	}); err != nil {
		t.Fatalf("CreateSession(%q): %v", token, err)
	}
}

// countAllSessions queries the raw table (including expired rows) so tests can
// verify that DeleteExpiredSessions actually removed rows.
func countAllSessions(t *testing.T, p purgerTestDB) int {
	t.Helper()
	var n int
	if err := p.raw.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&n); err != nil {
		t.Fatalf("COUNT sessions: %v", err)
	}
	return n
}

// TestDeleteExpiredSessions_RemovesExpired verifies that DeleteExpiredSessions
// removes only expired rows and leaves valid sessions intact.
func TestDeleteExpiredSessions_RemovesExpired(t *testing.T) {
	p := newPurgerTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	insertSession(t, p, "valid-1", now.Add(24*time.Hour))
	insertSession(t, p, "valid-2", now.Add(30*24*time.Hour))
	insertSession(t, p, "expired-1", now.Add(-time.Minute))
	insertSession(t, p, "expired-2", now.Add(-24*time.Hour))

	if got := countAllSessions(t, p); got != 4 {
		t.Fatalf("pre-purge: expected 4 sessions, got %d", got)
	}

	if err := p.q.DeleteExpiredSessions(ctx); err != nil {
		t.Fatalf("DeleteExpiredSessions: %v", err)
	}

	if got := countAllSessions(t, p); got != 2 {
		t.Errorf("post-purge: expected 2 sessions, got %d", got)
	}
}

// TestDeleteExpiredSessions_EmptyTable verifies that DeleteExpiredSessions is
// a no-op (no error) when no sessions exist.
func TestDeleteExpiredSessions_EmptyTable(t *testing.T) {
	p := newPurgerTestDB(t)
	if err := p.q.DeleteExpiredSessions(context.Background()); err != nil {
		t.Errorf("DeleteExpiredSessions on empty table: %v", err)
	}
}

// TestDeleteExpiredSessions_AllExpired verifies that all rows are removed when
// every session is expired.
func TestDeleteExpiredSessions_AllExpired(t *testing.T) {
	p := newPurgerTestDB(t)
	past := time.Now().UTC().Add(-time.Minute)
	insertSession(t, p, "a", past)
	insertSession(t, p, "b", past)

	if err := p.q.DeleteExpiredSessions(context.Background()); err != nil {
		t.Fatalf("DeleteExpiredSessions: %v", err)
	}
	if got := countAllSessions(t, p); got != 0 {
		t.Errorf("expected 0 sessions after purge, got %d", got)
	}
}
