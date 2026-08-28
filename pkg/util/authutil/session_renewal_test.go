package authutil_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/authutil"
	_ "modernc.org/sqlite"
)

// Sliding session expiry (#1647). expires_at used to be stamped once at login
// and never written again, so a user who opened the app every day was still
// forced back to the login screen on day 31.
//
// These tests drive authutil.SetNow rather than sleeping, and none of them may
// run in parallel — the clock is package state.

// newRenewalTestDB returns both handles: the raw *sql.DB to backdate session
// rows into shapes login cannot produce, and the queries the code under test
// uses.
//
// Note for anyone adding cases here: GetSession filters with
// `expires_at > datetime('now')`, which compares a Go-formatted local
// timestamp against SQLite's UTC one as plain text. That only sorts correctly
// when the two differ in the date part, so keep expiries at least a day either
// side of now rather than minutes.
func newRenewalTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	// A shared-cache memory DB, uniquely named per test: plain ":memory:" gives
	// every pooled connection its own empty database, and these tests read
	// through the pool while the code under test holds its own connection.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	const schema = `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			recovery_phrase_hash TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			is_admin INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			last_used_at DATETIME NOT NULL DEFAULT '1970-01-01 00:00:00',
			FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
		);
	`
	if _, err := sqlDB.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		t.Fatalf("get connection: %v", err)
	}
	return sqlDB, db.New(conn)
}

// newSignedInUser sets up the first user and returns their raw session token.
func newSignedInUser(t *testing.T, queries *db.Queries) string {
	t.Helper()
	res, err := authutil.Setup(context.Background(), queries, authutil.SetupParams{
		Username: "testuser",
		Password: "testpassword",
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	return res.SessionToken
}

// digest mirrors how sessions are keyed at rest — the raw token is never stored.
func digest(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

type sessionRow struct {
	createdAt  time.Time
	expiresAt  time.Time
	lastUsedAt time.Time
}

func readSession(t *testing.T, sqlDB *sql.DB, rawToken string) sessionRow {
	t.Helper()
	var row sessionRow
	err := sqlDB.QueryRow(
		`SELECT created_at, expires_at, last_used_at FROM sessions WHERE token = ?`,
		digest(rawToken),
	).Scan(&row.createdAt, &row.expiresAt, &row.lastUsedAt)
	if err != nil {
		t.Fatalf("read session: %v", err)
	}
	return row
}

// backdate rewrites a session into a shape login cannot produce — an old
// session, one near its cap, one already dead.
func backdate(t *testing.T, sqlDB *sql.DB, rawToken string, createdAt, expiresAt, lastUsedAt time.Time) {
	t.Helper()
	res, err := sqlDB.Exec(
		`UPDATE sessions SET created_at = ?, expires_at = ?, last_used_at = ? WHERE token = ?`,
		createdAt, expiresAt, lastUsedAt, digest(rawToken),
	)
	if err != nil {
		t.Fatalf("backdate session: %v", err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		t.Fatalf("backdate session: updated %d rows, want 1", n)
	}
}

// assertNear tolerates the sub-second drift between the pinned Go clock and
// what SQLite stores.
func assertNear(t *testing.T, got, want time.Time, what string) {
	t.Helper()
	if d := got.Sub(want); d > time.Second || d < -time.Second {
		t.Errorf("%s = %v, want ~%v (off by %v)", what, got, want, d)
	}
}

// The point of the whole change: using a session pushes its expiry out.
func TestValidateSession_RenewsExpiryOnUse(t *testing.T) {
	sqlDB, queries := newRenewalTestDB(t)
	token := newSignedInUser(t, queries)
	before := readSession(t, sqlDB, token)

	// Past the debounce window, so the next use renews.
	later := time.Now().Add(2 * time.Hour)
	defer authutil.SetNow(later)()

	if _, err := authutil.ValidateSession(context.Background(), queries, token); err != nil {
		t.Fatalf("validate: %v", err)
	}

	after := readSession(t, sqlDB, token)
	if !after.expiresAt.After(before.expiresAt) {
		t.Errorf("expires_at did not move: %v -> %v", before.expiresAt, after.expiresAt)
	}
	assertNear(t, after.expiresAt, later.Add(authutil.SessionDuration), "expires_at")
	assertNear(t, after.lastUsedAt, later, "last_used_at")
}

// Renewing on every request would make each authenticated call a write.
func TestValidateSession_DebouncesRenewal(t *testing.T) {
	sqlDB, queries := newRenewalTestDB(t)
	token := newSignedInUser(t, queries)
	before := readSession(t, sqlDB, token)

	// Well inside the debounce window.
	defer authutil.SetNow(time.Now().Add(authutil.SessionRenewInterval / 2))()

	for range 3 {
		if _, err := authutil.ValidateSession(context.Background(), queries, token); err != nil {
			t.Fatalf("validate: %v", err)
		}
	}

	after := readSession(t, sqlDB, token)
	assertNear(t, after.expiresAt, before.expiresAt, "expires_at")
	assertNear(t, after.lastUsedAt, before.lastUsedAt, "last_used_at")
}

// "Stay signed in" must not mean "signed in forever": renewal is clamped to
// sessionMaxLifetime past creation, so a leaked token still dies on schedule.
func TestValidateSession_ClampsRenewalToMaxLifetime(t *testing.T) {
	sqlDB, queries := newRenewalTestDB(t)
	token := newSignedInUser(t, queries)

	// A long-lived session near its cap. expires_at stays in the future so
	// GetSession still finds it.
	pinned := time.Now()
	createdAt := pinned.Add(-authutil.SessionMaxLifetime + 24*time.Hour)
	backdate(t, sqlDB, token, createdAt, pinned.Add(24*time.Hour), pinned.Add(-48*time.Hour))

	defer authutil.SetNow(pinned)()

	if _, err := authutil.ValidateSession(context.Background(), queries, token); err != nil {
		t.Fatalf("validate: %v", err)
	}

	after := readSession(t, sqlDB, token)
	cap := createdAt.Add(authutil.SessionMaxLifetime)
	assertNear(t, after.expiresAt, cap, "expires_at")
	if unclamped := pinned.Add(authutil.SessionDuration); !after.expiresAt.Before(unclamped) {
		t.Errorf("expires_at %v was not clamped below the unclamped %v", after.expiresAt, unclamped)
	}
}

// At the cap the expiry stops moving, and we stop writing the same value.
func TestValidateSession_StopsRenewingAtCap(t *testing.T) {
	sqlDB, queries := newRenewalTestDB(t)
	token := newSignedInUser(t, queries)

	// Already pinned to its cap, with the cap still a couple of days out — see
	// the note on newRenewalTestDB about staying clear of same-day expiries.
	pinned := time.Now()
	createdAt := pinned.Add(-authutil.SessionMaxLifetime + 48*time.Hour)
	staleUse := pinned.Add(-48 * time.Hour)
	backdate(t, sqlDB, token, createdAt, createdAt.Add(authutil.SessionMaxLifetime), staleUse)
	before := readSession(t, sqlDB, token)

	defer authutil.SetNow(pinned)()

	if _, err := authutil.ValidateSession(context.Background(), queries, token); err != nil {
		t.Fatalf("validate: %v", err)
	}

	after := readSession(t, sqlDB, token)
	assertNear(t, after.expiresAt, before.expiresAt, "expires_at")
	// last_used_at untouched proves no write happened at all, rather than a
	// write that happened to store the same expiry.
	assertNear(t, after.lastUsedAt, staleUse, "last_used_at")
}

// Renewal must never resurrect a dead session: GetSession filters it out
// before renewal is reached.
func TestValidateSession_ExpiredSessionStaysExpired(t *testing.T) {
	sqlDB, queries := newRenewalTestDB(t)
	token := newSignedInUser(t, queries)

	pinned := time.Now()
	expired := pinned.Add(-48 * time.Hour)
	backdate(t, sqlDB, token, pinned.Add(-31*24*time.Hour), expired, pinned.Add(-72*time.Hour))

	defer authutil.SetNow(pinned)()

	if _, err := authutil.ValidateSession(context.Background(), queries, token); err == nil {
		t.Fatal("expected an expired session to be rejected")
	}

	after := readSession(t, sqlDB, token)
	assertNear(t, after.expiresAt, expired, "expires_at")
}

// A session nobody uses still expires on the original fixed window.
func TestNewSession_UnusedSessionKeepsFixedWindow(t *testing.T) {
	sqlDB, queries := newRenewalTestDB(t)

	pinned := time.Now()
	restore := authutil.SetNow(pinned)
	token := newSignedInUser(t, queries)
	restore()

	row := readSession(t, sqlDB, token)
	assertNear(t, row.expiresAt, pinned.Add(authutil.SessionDuration), "expires_at")
	// A new session counts as just used, so the first renewal is a full
	// interval away rather than firing on the very next request.
	assertNear(t, row.lastUsedAt, pinned, "last_used_at")
}
