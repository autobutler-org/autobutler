package v0_albums

import (
	"context"
	"database/sql"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	_ "modernc.org/sqlite"
)

// --- buildTree (pure function, no DB needed) ---

func ptr64(v int64) *int64 { return &v }

func TestBuildTree_EmptyInput(t *testing.T) {
	result := buildTree(nil)
	if len(result) != 0 {
		t.Errorf("expected empty, got %d items", len(result))
	}
}

func TestBuildTree_AllRoots(t *testing.T) {
	albums := []AlbumJSON{
		{ID: 1, Name: "Alpha"},
		{ID: 2, Name: "Beta"},
		{ID: 3, Name: "Gamma"},
	}
	roots := buildTree(albums)
	if len(roots) != 3 {
		t.Fatalf("expected 3 roots, got %d", len(roots))
	}
}

func TestBuildTree_TwoLevels(t *testing.T) {
	albums := []AlbumJSON{
		{ID: 1, Name: "Parent"},
		{ID: 2, Name: "Child", ParentID: ptr64(1)},
	}
	roots := buildTree(albums)
	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}
	if len(roots[0].Children) != 1 || roots[0].Children[0].Name != "Child" {
		t.Errorf("expected Child under Parent, got %+v", roots[0].Children)
	}
}

func TestBuildTree_ThreeLevels(t *testing.T) {
	albums := []AlbumJSON{
		{ID: 1, Name: "Grandparent"},
		{ID: 2, Name: "Parent", ParentID: ptr64(1)},
		{ID: 3, Name: "Child", ParentID: ptr64(2)},
	}
	roots := buildTree(albums)
	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}
	if len(roots[0].Children) != 1 {
		t.Fatalf("expected 1 child of root, got %d", len(roots[0].Children))
	}
	if len(roots[0].Children[0].Children) != 1 {
		t.Fatalf("expected 1 grandchild, got %d", len(roots[0].Children[0].Children))
	}
	if roots[0].Children[0].Children[0].Name != "Child" {
		t.Errorf("unexpected grandchild name: %q", roots[0].Children[0].Children[0].Name)
	}
}

func TestBuildTree_MultipleChildrenPerParent(t *testing.T) {
	albums := []AlbumJSON{
		{ID: 1, Name: "Root"},
		{ID: 2, Name: "A", ParentID: ptr64(1)},
		{ID: 3, Name: "B", ParentID: ptr64(1)},
		{ID: 4, Name: "C", ParentID: ptr64(1)},
	}
	roots := buildTree(albums)
	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}
	if len(roots[0].Children) != 3 {
		t.Errorf("expected 3 children, got %d", len(roots[0].Children))
	}
}

func TestBuildTree_OrphanDropped(t *testing.T) {
	// Child whose parent doesn't exist: dropped (not a root, parent not found).
	albums := []AlbumJSON{
		{ID: 1, Name: "Root"},
		{ID: 2, Name: "Orphan", ParentID: ptr64(999)},
	}
	roots := buildTree(albums)
	if len(roots) != 1 {
		t.Errorf("expected 1 root (orphan dropped), got %d", len(roots))
	}
}

// --- Album SQL query layer ---

const albumSchema = `
CREATE TABLE IF NOT EXISTS photo_albums (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    parent_id INTEGER,
    smart_type TEXT,
    retention_days INTEGER,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_photo_albums_smart_type
    ON photo_albums (smart_type) WHERE smart_type IS NOT NULL;

CREATE TABLE IF NOT EXISTS photo_album_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    album_id INTEGER NOT NULL,
    device_serial TEXT NOT NULL DEFAULT '',
    rel_path TEXT NOT NULL,
    added_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (album_id) REFERENCES photo_albums (id) ON DELETE CASCADE,
    UNIQUE (album_id, device_serial, rel_path)
);
`

func newAlbumDB(t *testing.T) *db.Queries {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	if _, err := sqlDB.Exec(albumSchema); err != nil {
		t.Fatalf("schema: %v", err)
	}
	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		t.Fatalf("conn: %v", err)
	}
	return db.New(conn)
}

func TestCreateAndListAlbums(t *testing.T) {
	q := newAlbumDB(t)
	ctx := context.Background()

	a, err := q.CreateAlbum(ctx, db.CreateAlbumParams{Name: "Vacation"})
	if err != nil {
		t.Fatalf("CreateAlbum: %v", err)
	}
	if a.Name != "Vacation" {
		t.Errorf("expected 'Vacation', got %q", a.Name)
	}

	all, err := q.ListAlbums(ctx)
	if err != nil {
		t.Fatalf("ListAlbums: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 album, got %d", len(all))
	}
}

func TestDeleteAlbum(t *testing.T) {
	q := newAlbumDB(t)
	ctx := context.Background()

	a, _ := q.CreateAlbum(ctx, db.CreateAlbumParams{Name: "ToDelete"})
	if err := q.DeleteAlbum(ctx, a.ID); err != nil {
		t.Fatalf("DeleteAlbum: %v", err)
	}
	all, _ := q.ListAlbums(ctx)
	if len(all) != 0 {
		t.Errorf("expected 0 albums after delete, got %d", len(all))
	}
}

func TestRenameAlbum(t *testing.T) {
	q := newAlbumDB(t)
	ctx := context.Background()

	a, _ := q.CreateAlbum(ctx, db.CreateAlbumParams{Name: "Old"})
	renamed, err := q.RenameAlbum(ctx, db.RenameAlbumParams{Name: "New", ID: a.ID})
	if err != nil {
		t.Fatalf("RenameAlbum: %v", err)
	}
	if renamed.Name != "New" {
		t.Errorf("expected 'New', got %q", renamed.Name)
	}
}

func TestAddAndListAlbumItems(t *testing.T) {
	q := newAlbumDB(t)
	ctx := context.Background()

	album, _ := q.CreateAlbum(ctx, db.CreateAlbumParams{Name: "Beach"})
	item, err := q.AddPhotoToAlbum(ctx, db.AddPhotoToAlbumParams{
		AlbumID:      album.ID,
		DeviceSerial: "sda1",
		RelPath:      "photos/sunset.jpg",
	})
	if err != nil {
		t.Fatalf("AddPhotoToAlbum: %v", err)
	}
	if item.RelPath != "photos/sunset.jpg" {
		t.Errorf("expected relPath 'photos/sunset.jpg', got %q", item.RelPath)
	}

	items, _ := q.ListAlbumItems(ctx, album.ID)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

func TestAddPhotoToAlbum_Idempotent(t *testing.T) {
	q := newAlbumDB(t)
	ctx := context.Background()

	album, _ := q.CreateAlbum(ctx, db.CreateAlbumParams{Name: "Summer"})
	params := db.AddPhotoToAlbumParams{AlbumID: album.ID, RelPath: "pic.jpg"}
	if _, err := q.AddPhotoToAlbum(ctx, params); err != nil {
		t.Fatalf("first add: %v", err)
	}
	if _, err := q.AddPhotoToAlbum(ctx, params); err != nil {
		t.Fatalf("second add (should be idempotent): %v", err)
	}
	count, _ := q.CountAlbumItems(ctx, album.ID)
	if count != 1 {
		t.Errorf("expected 1 item after idempotent add, got %d", count)
	}
}

func TestRemovePhotoFromAlbum(t *testing.T) {
	q := newAlbumDB(t)
	ctx := context.Background()

	album, _ := q.CreateAlbum(ctx, db.CreateAlbumParams{Name: "Winter"})
	q.AddPhotoToAlbum(ctx, db.AddPhotoToAlbumParams{AlbumID: album.ID, RelPath: "snow.jpg"})
	if err := q.RemovePhotoFromAlbum(ctx, db.RemovePhotoFromAlbumParams{
		AlbumID: album.ID, RelPath: "snow.jpg",
	}); err != nil {
		t.Fatalf("RemovePhotoFromAlbum: %v", err)
	}
	count, _ := q.CountAlbumItems(ctx, album.ID)
	if count != 0 {
		t.Errorf("expected 0 items after remove, got %d", count)
	}
}

func TestDeleteAlbum_CascadesItems(t *testing.T) {
	q := newAlbumDB(t)
	ctx := context.Background()

	album, _ := q.CreateAlbum(ctx, db.CreateAlbumParams{Name: "Cascade"})
	q.AddPhotoToAlbum(ctx, db.AddPhotoToAlbumParams{AlbumID: album.ID, RelPath: "a.jpg"})
	q.AddPhotoToAlbum(ctx, db.AddPhotoToAlbumParams{AlbumID: album.ID, RelPath: "b.jpg"})

	if err := q.DeleteAlbum(ctx, album.ID); err != nil {
		t.Fatalf("DeleteAlbum: %v", err)
	}
	items, _ := q.ListAlbumItems(ctx, album.ID)
	if len(items) != 0 {
		t.Errorf("expected 0 items after cascade delete, got %d", len(items))
	}
}

func TestCountAlbumItems(t *testing.T) {
	q := newAlbumDB(t)
	ctx := context.Background()

	album, _ := q.CreateAlbum(ctx, db.CreateAlbumParams{Name: "Count"})
	for i, path := range []string{"a.jpg", "b.jpg", "c.jpg"} {
		q.AddPhotoToAlbum(ctx, db.AddPhotoToAlbumParams{AlbumID: album.ID, RelPath: path})
		count, _ := q.CountAlbumItems(ctx, album.ID)
		if count != int64(i+1) {
			t.Errorf("after %d adds: expected %d, got %d", i+1, i+1, count)
		}
	}
}

func TestListRootAlbums(t *testing.T) {
	q := newAlbumDB(t)
	ctx := context.Background()

	root, _ := q.CreateAlbum(ctx, db.CreateAlbumParams{Name: "Root"})
	q.CreateAlbum(ctx, db.CreateAlbumParams{
		Name:     "Child",
		ParentID: sql.NullInt64{Int64: root.ID, Valid: true},
	})

	roots, err := q.ListRootAlbums(ctx)
	if err != nil {
		t.Fatalf("ListRootAlbums: %v", err)
	}
	if len(roots) != 1 || roots[0].Name != "Root" {
		t.Errorf("expected 1 root album 'Root', got %+v", roots)
	}
}

func TestListChildAlbums(t *testing.T) {
	q := newAlbumDB(t)
	ctx := context.Background()

	parent, _ := q.CreateAlbum(ctx, db.CreateAlbumParams{Name: "Parent"})
	q.CreateAlbum(ctx, db.CreateAlbumParams{Name: "C1", ParentID: sql.NullInt64{Int64: parent.ID, Valid: true}})
	q.CreateAlbum(ctx, db.CreateAlbumParams{Name: "C2", ParentID: sql.NullInt64{Int64: parent.ID, Valid: true}})

	children, err := q.ListChildAlbums(ctx, sql.NullInt64{Int64: parent.ID, Valid: true})
	if err != nil {
		t.Fatalf("ListChildAlbums: %v", err)
	}
	if len(children) != 2 {
		t.Errorf("expected 2 children, got %d", len(children))
	}
}
