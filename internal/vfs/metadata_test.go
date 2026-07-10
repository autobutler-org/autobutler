package vfs_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/autobutler-org/autobutler/internal/vfs"
)

const testSchema = `
CREATE TABLE IF NOT EXISTS vfs_metadata (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    namespace   TEXT NOT NULL,
    path        TEXT NOT NULL,
    key         TEXT NOT NULL,
    value       TEXT NOT NULL,
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE (namespace, path, key)
);
CREATE INDEX IF NOT EXISTS idx_vfs_metadata_ns_path ON vfs_metadata (namespace, path);
CREATE INDEX IF NOT EXISTS idx_vfs_metadata_ns_key  ON vfs_metadata (namespace, key);

CREATE TABLE IF NOT EXISTS vfs_db_entries (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    namespace   TEXT NOT NULL,
    path        TEXT NOT NULL,
    is_dir      BOOLEAN NOT NULL DEFAULT 0,
    size        INTEGER NOT NULL DEFAULT 0,
    mime_type   TEXT NOT NULL DEFAULT '',
    content     BLOB,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE (namespace, path)
);
CREATE INDEX IF NOT EXISTS idx_vfs_db_entries_ns_path ON vfs_db_entries (namespace, path);
`

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if _, err := db.Exec(testSchema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func jsonVal(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return json.RawMessage(b)
}

func TestMetadataStore_GetEmpty(t *testing.T) {
	db := newTestDB(t)
	store := vfs.NewSQLiteMetadataStore(db)
	ctx := context.Background()

	m, err := store.Get(ctx, "photos", "/albums/foo/")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(m) != 0 {
		t.Fatalf("expected empty map, got %v", m)
	}
}

func TestMetadataStore_SetThenGet(t *testing.T) {
	db := newTestDB(t)
	store := vfs.NewSQLiteMetadataStore(db)
	ctx := context.Background()

	kv := map[string]json.RawMessage{
		"photos.rotation": jsonVal(t, 90),
		"files.tag":       jsonVal(t, "summer"),
	}
	if err := store.Set(ctx, "photos", "/albums/summer/", kv); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := store.Get(ctx, "photos", "/albums/summer/")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 keys, got %d: %v", len(got), got)
	}

	var rot int
	if err := json.Unmarshal(got["photos.rotation"], &rot); err != nil || rot != 90 {
		t.Fatalf("photos.rotation: got %v, want 90", got["photos.rotation"])
	}
	var tag string
	if err := json.Unmarshal(got["files.tag"], &tag); err != nil || tag != "summer" {
		t.Fatalf("files.tag: got %v, want summer", got["files.tag"])
	}
}

func TestMetadataStore_SetMerges(t *testing.T) {
	db := newTestDB(t)
	store := vfs.NewSQLiteMetadataStore(db)
	ctx := context.Background()

	if err := store.Set(ctx, "photos", "/p/", map[string]json.RawMessage{
		"key1": jsonVal(t, "v1"),
		"key2": jsonVal(t, "v2"),
	}); err != nil {
		t.Fatalf("Set 1: %v", err)
	}

	// Overwrite key1, add key3 — key2 should survive.
	if err := store.Set(ctx, "photos", "/p/", map[string]json.RawMessage{
		"key1": jsonVal(t, "v1-new"),
		"key3": jsonVal(t, "v3"),
	}); err != nil {
		t.Fatalf("Set 2: %v", err)
	}

	got, err := store.Get(ctx, "photos", "/p/")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 keys, got %d: %v", len(got), got)
	}
	var v1 string
	json.Unmarshal(got["key1"], &v1) //nolint:errcheck
	if v1 != "v1-new" {
		t.Fatalf("key1 not updated: %v", v1)
	}
	if _, ok := got["key2"]; !ok {
		t.Fatalf("key2 was lost after merge")
	}
	if _, ok := got["key3"]; !ok {
		t.Fatalf("key3 not set")
	}
}

func TestMetadataStore_DeleteKeys(t *testing.T) {
	db := newTestDB(t)
	store := vfs.NewSQLiteMetadataStore(db)
	ctx := context.Background()

	if err := store.Set(ctx, "photos", "/q/", map[string]json.RawMessage{
		"k1": jsonVal(t, 1),
		"k2": jsonVal(t, 2),
		"k3": jsonVal(t, 3),
	}); err != nil {
		t.Fatalf("Set: %v", err)
	}

	if err := store.DeleteKeys(ctx, "photos", "/q/", []string{"k1", "k3"}); err != nil {
		t.Fatalf("DeleteKeys: %v", err)
	}

	got, err := store.Get(ctx, "photos", "/q/")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 key, got %d: %v", len(got), got)
	}
	if _, ok := got["k2"]; !ok {
		t.Fatalf("k2 should remain")
	}
}

func TestMetadataStore_DeleteNonExistentKeyIsNoop(t *testing.T) {
	db := newTestDB(t)
	store := vfs.NewSQLiteMetadataStore(db)
	ctx := context.Background()

	// Should not error even if key doesn't exist.
	if err := store.DeleteKeys(ctx, "photos", "/nope/", []string{"missing"}); err != nil {
		t.Fatalf("DeleteKeys on non-existent: %v", err)
	}
}

func TestMetadataStore_QueryWithValue(t *testing.T) {
	db := newTestDB(t)
	store := vfs.NewSQLiteMetadataStore(db)
	ctx := context.Background()

	// Set the same key on two paths with different values.
	if err := store.Set(ctx, "photos", "/a/", map[string]json.RawMessage{
		"photos.rotation": jsonVal(t, 90),
	}); err != nil {
		t.Fatalf("Set a: %v", err)
	}
	if err := store.Set(ctx, "photos", "/b/", map[string]json.RawMessage{
		"photos.rotation": jsonVal(t, 180),
	}); err != nil {
		t.Fatalf("Set b: %v", err)
	}
	if err := store.Set(ctx, "photos", "/c/", map[string]json.RawMessage{
		"photos.rotation": jsonVal(t, 90),
	}); err != nil {
		t.Fatalf("Set c: %v", err)
	}

	// Query for rotation=90 → should return /a/ and /c/.
	entries, err := store.Query(ctx, "photos", "photos.rotation", jsonVal(t, 90))
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(entries), entries)
	}
	paths := map[string]bool{entries[0].Path: true, entries[1].Path: true}
	if !paths["/a/"] || !paths["/c/"] {
		t.Fatalf("unexpected paths: %v", paths)
	}
}

func TestMetadataStore_QueryNilValue(t *testing.T) {
	db := newTestDB(t)
	store := vfs.NewSQLiteMetadataStore(db)
	ctx := context.Background()

	if err := store.Set(ctx, "photos", "/x/", map[string]json.RawMessage{
		"photos.starred": jsonVal(t, true),
	}); err != nil {
		t.Fatalf("Set x: %v", err)
	}
	if err := store.Set(ctx, "photos", "/y/", map[string]json.RawMessage{
		"photos.starred": jsonVal(t, false),
	}); err != nil {
		t.Fatalf("Set y: %v", err)
	}
	if err := store.Set(ctx, "photos", "/z/", map[string]json.RawMessage{
		"other.key": jsonVal(t, "unrelated"),
	}); err != nil {
		t.Fatalf("Set z: %v", err)
	}

	// nil value → existence check → should find /x/ and /y/ (both have photos.starred set).
	entries, err := store.Query(ctx, "photos", "photos.starred", nil)
	if err != nil {
		t.Fatalf("Query nil: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	for _, e := range entries {
		if e.Path == "/z/" {
			t.Fatalf("/z/ should not be in results (doesn't have photos.starred)")
		}
	}
}
