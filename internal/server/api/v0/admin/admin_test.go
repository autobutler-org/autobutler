package v0_admin

import (
	"context"
	"database/sql"
	"testing"

	"github.com/autobutler-org/autobutler/internal/db"
	_ "modernc.org/sqlite"
)

// roleSchema mirrors migrations 002_auth.up + 015_user_roles.up for in-memory tests.
const roleSchema = `
CREATE TABLE IF NOT EXISTS users (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    username             TEXT NOT NULL UNIQUE,
    password_hash        TEXT NOT NULL,
    recovery_phrase_hash TEXT NOT NULL,
    created_at           DATETIME NOT NULL DEFAULT (datetime('now')),
    role                 TEXT NOT NULL DEFAULT 'owner'
        CHECK (role IN ('owner', 'admin', 'user'))
);
`

func newTestRoleDB(t *testing.T) *db.Queries {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	if _, err := sqlDB.Exec(roleSchema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		t.Fatalf("get connection: %v", err)
	}
	return db.New(conn)
}

func insertUser(t *testing.T, q *db.Queries, username, role string) int64 {
	t.Helper()
	ctx := context.Background()
	// Insert directly; not using CreateUser query to set role in the same INSERT.
	sqlDB, _ := sql.Open("sqlite", ":memory:") // unused; we reuse the passed q
	_ = sqlDB
	// Use ListUserRoles to find by username after inserting via raw exec.
	// We bypass sqlc here because CreateUser doesn't set role.
	// Use the underlying connection via a workaround: run SQL via db.Queries' parent.
	// Simpler: just call CreateUser then SetUserRole.
	user, err := q.CreateUser(ctx, db.CreateUserParams{
		Username:           username,
		PasswordHash:       "hash",
		RecoveryPhraseHash: "phrase",
	})
	if err != nil {
		t.Fatalf("CreateUser(%s): %v", username, err)
	}
	if role != "owner" {
		if err := q.SetUserRole(ctx, db.SetUserRoleParams{Role: role, ID: user.ID}); err != nil {
			t.Fatalf("SetUserRole(%s, %s): %v", username, role, err)
		}
	}
	return user.ID
}

func TestGetUserRole(t *testing.T) {
	q := newTestRoleDB(t)
	id := insertUser(t, q, "alice", "owner")

	role, err := q.GetUserRole(context.Background(), id)
	if err != nil {
		t.Fatalf("GetUserRole: %v", err)
	}
	if role != "owner" {
		t.Errorf("expected 'owner', got %q", role)
	}
}

func TestSetUserRole_Promote(t *testing.T) {
	q := newTestRoleDB(t)
	id := insertUser(t, q, "bob", "user")

	if err := q.SetUserRole(context.Background(), db.SetUserRoleParams{Role: "admin", ID: id}); err != nil {
		t.Fatalf("SetUserRole: %v", err)
	}

	role, _ := q.GetUserRole(context.Background(), id)
	if role != "admin" {
		t.Errorf("expected 'admin' after promote, got %q", role)
	}
}

func TestIsAdmin_Owner(t *testing.T) {
	q := newTestRoleDB(t)
	id := insertUser(t, q, "carol", "owner")

	isAdmin, err := q.IsAdmin(context.Background(), id)
	if err != nil {
		t.Fatalf("IsAdmin: %v", err)
	}
	if isAdmin != 1 {
		t.Errorf("owner should be admin-privileged, got %d", isAdmin)
	}
}

func TestIsAdmin_Admin(t *testing.T) {
	q := newTestRoleDB(t)
	id := insertUser(t, q, "dave", "admin")

	isAdmin, _ := q.IsAdmin(context.Background(), id)
	if isAdmin != 1 {
		t.Errorf("admin should be admin-privileged, got %d", isAdmin)
	}
}

func TestIsAdmin_User(t *testing.T) {
	q := newTestRoleDB(t)
	id := insertUser(t, q, "eve", "user")

	isAdmin, _ := q.IsAdmin(context.Background(), id)
	if isAdmin != 0 {
		t.Errorf("regular user should not be admin-privileged, got %d", isAdmin)
	}
}

func TestCountAdmins(t *testing.T) {
	q := newTestRoleDB(t)
	ctx := context.Background()

	// Initially zero admins
	count, err := q.CountAdmins(ctx)
	if err != nil {
		t.Fatalf("CountAdmins: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	insertUser(t, q, "owner1", "owner")  // owners count as admins
	insertUser(t, q, "admin1", "admin")  // admins too
	insertUser(t, q, "user1", "user")    // users do not

	count, _ = q.CountAdmins(ctx)
	if count != 2 {
		t.Errorf("expected 2 (owner + admin), got %d", count)
	}
}

func TestListUserRoles(t *testing.T) {
	q := newTestRoleDB(t)

	insertUser(t, q, "u1", "owner")
	insertUser(t, q, "u2", "admin")
	insertUser(t, q, "u3", "user")

	rows, err := q.ListUserRoles(context.Background())
	if err != nil {
		t.Fatalf("ListUserRoles: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	roleMap := make(map[string]string)
	for _, r := range rows {
		roleMap[r.Username] = r.Role
	}
	for username, expectedRole := range map[string]string{"u1": "owner", "u2": "admin", "u3": "user"} {
		if roleMap[username] != expectedRole {
			t.Errorf("user %s: expected role %q, got %q", username, expectedRole, roleMap[username])
		}
	}
}
