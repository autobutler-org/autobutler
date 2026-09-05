package dbtest_test

import (
	"testing"

	"github.com/autobutler-org/quark/internal/db/dbtest"
)

// The point of running the real migrations is that tests get the real
// constraints. If foreign keys were not enforced on this connection every test
// converted to NewDB would still pass while proving nothing, so assert it
// directly: an orphan insert must be rejected, and a cascade must run.
func TestNewDBEnforcesForeignKeys(t *testing.T) {
	database := dbtest.NewDB(t)

	if _, err := database.Db.Exec(
		`INSERT INTO sessions (token, user_id, expires_at) VALUES ('t', 999, datetime('now'))`,
	); err == nil {
		t.Fatal("inserted a session for a user that does not exist")
	}

	if _, err := database.Db.Exec(
		`INSERT INTO users (username, password_hash, recovery_phrase_hash) VALUES ('u', 'h', 'r')`,
	); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := database.Db.Exec(
		`INSERT INTO sessions (token, user_id, expires_at)
		 SELECT 't', id, datetime('now') FROM users WHERE username = 'u'`,
	); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if _, err := database.Db.Exec(`DELETE FROM users WHERE username = 'u'`); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	var sessions int
	if err := database.Db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&sessions); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if sessions != 0 {
		t.Fatalf("ON DELETE CASCADE did not run: %d sessions left", sessions)
	}
}
