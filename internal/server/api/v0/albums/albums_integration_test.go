package v0_albums_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autobutler-org/autobutler/internal/db"
	v0_albums "github.com/autobutler-org/autobutler/internal/server/api/v0/albums"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

// newAlbumsTestDB creates an in-memory SQLite DB with the photo_albums schema.
func newAlbumsTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	_, err = sqlDB.Exec(`
		CREATE TABLE IF NOT EXISTS photo_albums (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			parent_id INTEGER,
			smart_type TEXT,
			retention_days INTEGER,
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
			FOREIGN KEY (parent_id) REFERENCES photo_albums (id) ON DELETE CASCADE
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_photo_albums_smart_type
			ON photo_albums (smart_type) WHERE smart_type IS NOT NULL;
		CREATE TABLE IF NOT EXISTS photo_album_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			album_id INTEGER NOT NULL,
			device_serial TEXT NOT NULL,
			rel_path TEXT NOT NULL,
			added_at DATETIME NOT NULL DEFAULT (datetime('now')),
			FOREIGN KEY (album_id) REFERENCES photo_albums (id) ON DELETE CASCADE,
			UNIQUE (album_id, device_serial, rel_path)
		);
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}
	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		t.Fatalf("get connection: %v", err)
	}
	return sqlDB, db.New(conn)
}

func newAlbumsEngine(t *testing.T, sqlDB *sql.DB, queries *db.Queries) *gin.Engine {
	t.Helper()
	deps := deputil.NewDependencies().WithDatabase(&db.DatabaseSqlc{
		Db:      sqlDB,
		Queries: queries,
	})
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c = ctxutil.With(c, "deps", deps)
		c.Next()
	})
	group := engine.Group("/api/v0")
	serverutil.RegisterRouterWithGroup(group, v0_albums.NewRouter())
	return engine
}

func doAlbumReq(engine *gin.Engine, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

// TestBuildTree_Empty verifies buildTree returns an empty slice for no input.
func TestBuildTree_Empty(t *testing.T) {
	sqlDB, queries := newAlbumsTestDB(t)
	engine := newAlbumsEngine(t, sqlDB, queries)

	w := doAlbumReq(engine, http.MethodGet, "/api/v0/albums?tree=true")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var albums []v0_albums.AlbumJSON
	if err := json.Unmarshal(w.Body.Bytes(), &albums); err != nil {
		t.Fatalf("unmarshal: %v\nbody: %s", err, w.Body.String())
	}
	if len(albums) != 0 {
		t.Errorf("expected empty tree, got %d roots", len(albums))
	}
}

// TestListAlbums_Empty verifies GET /albums returns an empty array when no albums exist.
func TestListAlbums_Empty(t *testing.T) {
	sqlDB, queries := newAlbumsTestDB(t)
	engine := newAlbumsEngine(t, sqlDB, queries)

	w := doAlbumReq(engine, http.MethodGet, "/api/v0/albums")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var albums []v0_albums.AlbumJSON
	if err := json.Unmarshal(w.Body.Bytes(), &albums); err != nil {
		t.Fatalf("unmarshal: %v\nbody: %s", err, w.Body.String())
	}
	if albums == nil {
		t.Error("expected empty slice, not null")
	}
}

// TestCreateAndListAlbum verifies creating an album and listing it back.
func TestCreateAndListAlbum(t *testing.T) {
	sqlDB, queries := newAlbumsTestDB(t)
	engine := newAlbumsEngine(t, sqlDB, queries)
	ctx := context.Background()

	album, err := queries.CreateAlbum(ctx, db.CreateAlbumParams{Name: "Vacation 2025"})
	if err != nil {
		t.Fatalf("CreateAlbum: %v", err)
	}

	w := doAlbumReq(engine, http.MethodGet, "/api/v0/albums")
	if w.Code != http.StatusOK {
		t.Fatalf("list returned %d: %s", w.Code, w.Body.String())
	}

	var albums []v0_albums.AlbumJSON
	if err := json.Unmarshal(w.Body.Bytes(), &albums); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(albums) != 1 {
		t.Fatalf("expected 1 album, got %d", len(albums))
	}
	if albums[0].Name != "Vacation 2025" {
		t.Errorf("album name = %q; want 'Vacation 2025'", albums[0].Name)
	}
	if albums[0].ID != album.ID {
		t.Errorf("album ID = %d; want %d", albums[0].ID, album.ID)
	}
}

// TestListAlbums_TreeMode verifies ?tree=true nests child albums under parents.
func TestListAlbums_TreeMode(t *testing.T) {
	sqlDB, queries := newAlbumsTestDB(t)
	engine := newAlbumsEngine(t, sqlDB, queries)
	ctx := context.Background()

	parent, err := queries.CreateAlbum(ctx, db.CreateAlbumParams{Name: "Family"})
	if err != nil {
		t.Fatalf("CreateAlbum parent: %v", err)
	}
	_, err = queries.CreateAlbum(ctx, db.CreateAlbumParams{
		Name:     "Christmas",
		ParentID: sql.NullInt64{Int64: parent.ID, Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateAlbum child: %v", err)
	}

	w := doAlbumReq(engine, http.MethodGet, "/api/v0/albums?tree=true")
	if w.Code != http.StatusOK {
		t.Fatalf("tree returned %d: %s", w.Code, w.Body.String())
	}

	var tree []v0_albums.AlbumJSON
	if err := json.Unmarshal(w.Body.Bytes(), &tree); err != nil {
		t.Fatalf("unmarshal: %v\nbody: %s", err, w.Body.String())
	}
	if len(tree) != 1 {
		t.Fatalf("expected 1 root album, got %d", len(tree))
	}
	if tree[0].Name != "Family" {
		t.Errorf("root name = %q; want 'Family'", tree[0].Name)
	}
	if len(tree[0].Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(tree[0].Children))
	}
	if tree[0].Children[0].Name != "Christmas" {
		t.Errorf("child name = %q; want 'Christmas'", tree[0].Children[0].Name)
	}
}

// TestGetAlbum_NotFound verifies GET /albums/:id returns 404 for missing album.
func TestGetAlbum_NotFound(t *testing.T) {
	sqlDB, queries := newAlbumsTestDB(t)
	engine := newAlbumsEngine(t, sqlDB, queries)

	w := doAlbumReq(engine, http.MethodGet, "/api/v0/albums/99999")
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// TestDeleteAlbum verifies DELETE /albums/:id removes the album.
func TestDeleteAlbum(t *testing.T) {
	sqlDB, queries := newAlbumsTestDB(t)
	engine := newAlbumsEngine(t, sqlDB, queries)
	ctx := context.Background()

	album, err := queries.CreateAlbum(ctx, db.CreateAlbumParams{Name: "To Delete"})
	if err != nil {
		t.Fatalf("CreateAlbum: %v", err)
	}

	path := "/api/v0/albums/" + itoa(album.ID)
	w := doAlbumReq(engine, http.MethodDelete, path)
	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Fatalf("DELETE returned %d: %s", w.Code, w.Body.String())
	}

	w2 := doAlbumReq(engine, http.MethodGet, path)
	if w2.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d: %s", w2.Code, w2.Body.String())
	}
}

// TestListAlbums_ItemCount verifies itemCount is included in album listing.
func TestListAlbums_ItemCount(t *testing.T) {
	sqlDB, queries := newAlbumsTestDB(t)
	engine := newAlbumsEngine(t, sqlDB, queries)
	ctx := context.Background()

	album, _ := queries.CreateAlbum(ctx, db.CreateAlbumParams{Name: "With Photos"})
	for _, path := range []string{"photo1.jpg", "photo2.jpg"} {
		_, err := queries.AddPhotoToAlbum(ctx, db.AddPhotoToAlbumParams{
			AlbumID:      album.ID,
			DeviceSerial: "DEV-001",
			RelPath:      path,
		})
		if err != nil {
			t.Fatalf("AddPhotoToAlbum: %v", err)
		}
	}

	w := doAlbumReq(engine, http.MethodGet, "/api/v0/albums")
	if w.Code != http.StatusOK {
		t.Fatalf("list returned %d: %s", w.Code, w.Body.String())
	}

	var albums []v0_albums.AlbumJSON
	if err := json.Unmarshal(w.Body.Bytes(), &albums); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(albums) == 0 {
		t.Fatal("no albums in response")
	}
	if albums[0].ItemCount != 2 {
		t.Errorf("itemCount = %d; want 2", albums[0].ItemCount)
	}
}

// itoa converts int64 to string for URL construction.
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}
