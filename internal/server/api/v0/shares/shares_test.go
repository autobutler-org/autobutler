package v0_shares

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/autobutler-org/autobutler/internal/db"
	_ "modernc.org/sqlite"
)

const shareSchema = `
CREATE TABLE IF NOT EXISTS share_links (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    token          TEXT    NOT NULL UNIQUE,
    device_serial  TEXT    NOT NULL DEFAULT '',
    rel_path       TEXT    NOT NULL,
    password_hash  TEXT    NOT NULL DEFAULT '',
    max_uses       INTEGER NOT NULL DEFAULT 0,
    use_count      INTEGER NOT NULL DEFAULT 0,
    expires_at     DATETIME,
    created_at     DATETIME NOT NULL DEFAULT (datetime('now')),
    created_by     TEXT    NOT NULL DEFAULT ''
);
`

func newShareDB(t *testing.T) *db.Queries {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	if _, err := sqlDB.Exec(shareSchema); err != nil {
		t.Fatalf("schema: %v", err)
	}
	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		t.Fatalf("conn: %v", err)
	}
	return db.New(conn)
}

func createLink(t *testing.T, q *db.Queries, params db.CreateShareLinkParams) db.ShareLink {
	t.Helper()
	id, err := q.CreateShareLink(context.Background(), params)
	if err != nil {
		t.Fatalf("CreateShareLink: %v", err)
	}
	link, err := q.GetShareLinkByToken(context.Background(), params.Token)
	if err != nil {
		t.Fatalf("GetShareLinkByToken after create (id=%d): %v", id, err)
	}
	return link
}

// --- generateToken ---

func TestGenerateToken_Length(t *testing.T) {
	tok, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}
	// 32 raw bytes → base64url without padding ≈ 43 chars.
	if len(tok) < 40 {
		t.Errorf("token too short: %d chars", len(tok))
	}
}

func TestGenerateToken_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for range 20 {
		tok, err := generateToken()
		if err != nil {
			t.Fatalf("generateToken: %v", err)
		}
		if seen[tok] {
			t.Fatalf("duplicate token: %s", tok)
		}
		seen[tok] = true
	}
}

// --- CRUD ---

func TestCreateAndGetShareLink(t *testing.T) {
	q := newShareDB(t)
	link := createLink(t, q, db.CreateShareLinkParams{
		Token:     "tok1",
		RelPath:   "docs/report.abdoc",
		CreatedBy: "alice",
	})
	if link.Token != "tok1" {
		t.Errorf("token: got %q", link.Token)
	}
	if link.RelPath != "docs/report.abdoc" {
		t.Errorf("relPath: got %q", link.RelPath)
	}
	if link.UseCount != 0 {
		t.Errorf("useCount: got %d", link.UseCount)
	}
}

func TestGetShareLinkByToken_NotFound(t *testing.T) {
	q := newShareDB(t)
	_, err := q.GetShareLinkByToken(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing token, got nil")
	}
}

func TestDeleteShareLink(t *testing.T) {
	q := newShareDB(t)
	link := createLink(t, q, db.CreateShareLinkParams{Token: "del_me", RelPath: "x"})
	if err := q.DeleteShareLink(context.Background(), link.ID); err != nil {
		t.Fatalf("DeleteShareLink: %v", err)
	}
	_, err := q.GetShareLinkByToken(context.Background(), "del_me")
	if err == nil {
		t.Fatal("expected not-found after delete")
	}
}

func TestListShareLinks(t *testing.T) {
	q := newShareDB(t)
	for _, tok := range []string{"a", "b", "c"} {
		createLink(t, q, db.CreateShareLinkParams{Token: tok, RelPath: tok})
	}
	links, err := q.ListShareLinks(context.Background())
	if err != nil {
		t.Fatalf("ListShareLinks: %v", err)
	}
	if len(links) != 3 {
		t.Errorf("expected 3 links, got %d", len(links))
	}
}

// --- UseCount ---

func TestIncrementUseCount(t *testing.T) {
	q := newShareDB(t)
	createLink(t, q, db.CreateShareLinkParams{Token: "uc", RelPath: "f"})

	for i := range 3 {
		if err := q.IncrementShareLinkUseCount(context.Background(), "uc"); err != nil {
			t.Fatalf("increment %d: %v", i, err)
		}
	}
	link, _ := q.GetShareLinkByToken(context.Background(), "uc")
	if link.UseCount != 3 {
		t.Errorf("expected useCount=3, got %d", link.UseCount)
	}
}

// --- Expiry ---

func TestDeleteExpiredShareLinks(t *testing.T) {
	q := newShareDB(t)

	// Use UTC so SQLite's datetime('now') comparison is consistent.
	expired := sql.NullTime{Time: time.Now().UTC().Add(-time.Hour), Valid: true}
	notExpired := sql.NullTime{Time: time.Now().UTC().Add(time.Hour), Valid: true}

	createLink(t, q, db.CreateShareLinkParams{Token: "exp", RelPath: "e", ExpiresAt: expired})
	createLink(t, q, db.CreateShareLinkParams{Token: "live", RelPath: "l", ExpiresAt: notExpired})
	createLink(t, q, db.CreateShareLinkParams{Token: "perm", RelPath: "p"}) // no expiry

	if err := q.DeleteExpiredShareLinks(context.Background()); err != nil {
		t.Fatalf("DeleteExpiredShareLinks: %v", err)
	}

	links, _ := q.ListShareLinks(context.Background())
	tokens := make(map[string]bool)
	for _, l := range links {
		tokens[l.Token] = true
	}
	if tokens["exp"] {
		t.Error("expired link should have been deleted")
	}
	if !tokens["live"] {
		t.Error("non-expired link should still exist")
	}
	if !tokens["perm"] {
		t.Error("permanent link should still exist")
	}
}

// --- toShareLinkJSON ---

func TestToShareLinkJSON_NoExpiry(t *testing.T) {
	link := db.ShareLink{
		ID: 1, Token: "abc", RelPath: "photos/trip.jpg",
		CreatedAt: time.Now(), CreatedBy: "bob",
	}
	j := toShareLinkJSON(link, "https://butler.local")
	if j.ExpiresAt != nil {
		t.Errorf("expected nil ExpiresAt for permanent link")
	}
	if j.URL != "https://butler.local/s/abc" {
		t.Errorf("wrong URL: %s", j.URL)
	}
	if j.HasPassword {
		t.Error("HasPassword should be false when no hash")
	}
}

func TestToShareLinkJSON_WithPassword(t *testing.T) {
	link := db.ShareLink{
		ID: 2, Token: "xyz", RelPath: "docs/secret.abdoc",
		PasswordHash: "$2a$10$fakehash",
		CreatedAt:    time.Now(),
	}
	j := toShareLinkJSON(link, "https://butler.local")
	if !j.HasPassword {
		t.Error("HasPassword should be true when hash is set")
	}
}

func TestToShareLinkJSON_WithExpiry(t *testing.T) {
	exp := time.Now().Add(24 * time.Hour)
	link := db.ShareLink{
		ID:        3,
		Token:     "exp_tok",
		RelPath:   "file.txt",
		ExpiresAt: sql.NullTime{Time: exp, Valid: true},
		CreatedAt: time.Now(),
	}
	j := toShareLinkJSON(link, "https://butler.local")
	if j.ExpiresAt == nil {
		t.Fatal("ExpiresAt should be set")
	}
	parsed, err := time.Parse(time.RFC3339, *j.ExpiresAt)
	if err != nil {
		t.Fatalf("ExpiresAt parse: %v", err)
	}
	if parsed.Unix() != exp.UTC().Unix() {
		t.Errorf("ExpiresAt mismatch: got %s, want %s", parsed, exp.UTC())
	}
}
