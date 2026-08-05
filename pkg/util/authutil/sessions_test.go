package authutil_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/util/authutil"
	_ "modernc.org/sqlite"
)

// setupUserWithSessions creates a user and N sessions, returning the user ID
// and the raw session tokens so callers can compute expected IDs.
func setupUserWithSessions(t *testing.T, q *db.Queries, n int) (int64, []string) {
	t.Helper()
	ctx := context.Background()

	user, err := q.CreateUser(ctx, db.CreateUserParams{
		Username:           "testuser",
		PasswordHash:       "hash",
		RecoveryPhraseHash: "rphash",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	tokens := make([]string, n)
	for i := range n {
		tok, err := authutil.GenerateSessionToken()
		if err != nil {
			t.Fatalf("GenerateSessionToken: %v", err)
		}
		_, err = q.CreateSession(ctx, db.CreateSessionParams{
			Token:     tok,
			UserID:    user.ID,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})
		if err != nil {
			t.Fatalf("CreateSession: %v", err)
		}
		tokens[i] = tok
	}
	return user.ID, tokens
}

func TestListActiveSessions(t *testing.T) {
	q := newTestDB(t)
	ctx := context.Background()
	userID, _ := setupUserWithSessions(t, q, 3)

	sessions, err := authutil.ListActiveSessions(ctx, q, userID)
	if err != nil {
		t.Fatalf("ListActiveSessions: %v", err)
	}
	if len(sessions) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(sessions))
	}
	for _, s := range sessions {
		if s.ID == "" {
			t.Error("session ID should not be empty")
		}
		if s.ExpiresAt.IsZero() {
			t.Error("session ExpiresAt should not be zero")
		}
	}
}

func TestListActiveSessions_Empty(t *testing.T) {
	q := newTestDB(t)
	ctx := context.Background()
	userID, _ := setupUserWithSessions(t, q, 0)

	sessions, err := authutil.ListActiveSessions(ctx, q, userID)
	if err != nil {
		t.Fatalf("ListActiveSessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestRevokeSession(t *testing.T) {
	q := newTestDB(t)
	ctx := context.Background()
	userID, _ := setupUserWithSessions(t, q, 2)

	sessions, err := authutil.ListActiveSessions(ctx, q, userID)
	if err != nil {
		t.Fatalf("ListActiveSessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions before revoke, got %d", len(sessions))
	}

	targetID := sessions[0].ID
	deleted, err := authutil.RevokeSession(ctx, q, userID, targetID)
	if err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}
	if !deleted {
		t.Error("expected deleted=true")
	}

	remaining, err := authutil.ListActiveSessions(ctx, q, userID)
	if err != nil {
		t.Fatalf("ListActiveSessions after revoke: %v", err)
	}
	if len(remaining) != 1 {
		t.Errorf("expected 1 session after revoke, got %d", len(remaining))
	}
	if remaining[0].ID == targetID {
		t.Error("revoked session still present")
	}
}

func TestRevokeSession_NotFound(t *testing.T) {
	q := newTestDB(t)
	ctx := context.Background()
	userID, _ := setupUserWithSessions(t, q, 1)

	deleted, err := authutil.RevokeSession(ctx, q, userID, "nonexistent-hash")
	if err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}
	if deleted {
		t.Error("expected deleted=false for unknown session ID")
	}
}

func TestRevokeSession_WrongUser(t *testing.T) {
	q := newTestDB(t)
	ctx := context.Background()
	userID, _ := setupUserWithSessions(t, q, 1)

	sessions, _ := authutil.ListActiveSessions(ctx, q, userID)
	targetID := sessions[0].ID

	// Try to revoke with a different userID.
	deleted, err := authutil.RevokeSession(ctx, q, userID+99, targetID)
	if err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}
	if deleted {
		t.Error("should not delete another user's session")
	}
}

func TestRevokeAllSessions(t *testing.T) {
	q := newTestDB(t)
	ctx := context.Background()
	userID, _ := setupUserWithSessions(t, q, 4)

	if err := authutil.RevokeAllSessions(ctx, q, userID); err != nil {
		t.Fatalf("RevokeAllSessions: %v", err)
	}

	remaining, err := authutil.ListActiveSessions(ctx, q, userID)
	if err != nil {
		t.Fatalf("ListActiveSessions after revoke-all: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("expected 0 sessions after revoke-all, got %d", len(remaining))
	}
}

// newTestDBRaw is like newTestDB but also returns the underlying *sql.DB so
// tests can inspect raw table contents (e.g. verify tokens are hashed at rest).
// SetMaxOpenConns(1) forces a single shared connection so the same in-memory
// SQLite database is visible to both Queries and raw *sql.DB calls.
func newTestDBRaw(t *testing.T) (*db.Queries, *sql.DB) {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	// Single connection so the same in-memory database is shared between
	// db.Queries calls and raw *sql.DB inspection.
	sqlDB.SetMaxOpenConns(1)

	schema := `
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
	if _, err := sqlDB.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	return db.New(sqlDB), sqlDB
}

// TestPurgeExpiredSessions verifies that expired sessions are removed while
// valid sessions survive.
func TestPurgeExpiredSessions(t *testing.T) {
	queries, rawDB := newTestDBRaw(t)
	ctx := context.Background()

	_, err := rawDB.ExecContext(ctx,
		`INSERT INTO users (username, password_hash, recovery_phrase_hash)
		 VALUES ('admin', 'hash', 'rphash')`)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	var userID int64
	if err := rawDB.QueryRowContext(ctx,
		`SELECT id FROM users WHERE username = 'admin'`).Scan(&userID); err != nil {
		t.Fatalf("get user id: %v", err)
	}

	// Expired session.
	if _, err = rawDB.ExecContext(ctx,
		`INSERT INTO sessions (token, user_id, expires_at)
		 VALUES ('expiredhash', ?, datetime('now', '-1 day'))`, userID); err != nil {
		t.Fatalf("insert expired session: %v", err)
	}
	// Valid session.
	if _, err = rawDB.ExecContext(ctx,
		`INSERT INTO sessions (token, user_id, expires_at)
		 VALUES ('validhash', ?, datetime('now', '+30 days'))`, userID); err != nil {
		t.Fatalf("insert valid session: %v", err)
	}

	if err := authutil.PurgeExpiredSessions(ctx, queries); err != nil {
		t.Fatalf("PurgeExpiredSessions: %v", err)
	}

	var count int
	if err := rawDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions`).Scan(&count); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 session after purge, got %d", count)
	}

	var remaining string
	if err := rawDB.QueryRowContext(ctx, `SELECT token FROM sessions LIMIT 1`).Scan(&remaining); err != nil {
		t.Fatalf("select remaining: %v", err)
	}
	if remaining != "validhash" {
		t.Errorf("expected 'validhash' to survive purge, got %q", remaining)
	}
}
