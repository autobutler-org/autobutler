package v0_albums_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
	v0_albums "github.com/autobutler-org/quark/internal/server/api/v0/albums"
	"github.com/autobutler-org/quark/pkg/util/accessutil"
	"github.com/autobutler-org/quark/pkg/util/albumutil"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/eventbus"
	"github.com/autobutler-org/quark/pkg/util/favoritesutil"
	"github.com/autobutler-org/quark/pkg/util/fileutil"
	"github.com/autobutler-org/quark/pkg/util/photoutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
	"github.com/autobutler-org/quark/pkg/vfs"
	"github.com/gin-gonic/gin"
)

// pointerFixture is an internal drive under a temporary HOME, a database, and
// the albums API mounted on a real gin engine, acting as one account (#3488).
// Albums hold pointers to files: these tests change the files underneath them
// through the same services the Files routes call and read the albums back
// over HTTP.
type pointerFixture struct {
	t         *testing.T
	ctx       context.Context
	database  *db.DatabaseSqlc
	registry  vfs.Registry
	fsys      vfs.VFS
	engine    *gin.Engine
	owner     int64
	principal *accessutil.Principal
}

func newPointerFixture(t *testing.T, files ...string) *pointerFixture {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	if _, err := storageutil.GetFilesDir(); err != nil {
		t.Fatal(err)
	}
	storage := storageutil.NewStorageService(systemDevice{})
	registry := vfs.NewRegistry()
	namespace := vfs.FilesNamespace("")
	if err := registry.Register(vfs.Namespace{ID: namespace}, vfs.NewStorageServiceVFS(storage, namespace)); err != nil {
		t.Fatal(err)
	}
	fsys, err := fileutil.FilesVFS(registry, "")
	if err != nil {
		t.Fatal(err)
	}
	f := &pointerFixture{
		t:        t,
		ctx:      context.Background(),
		database: dbtest.NewDB(t),
		registry: registry,
		fsys:     fsys,
	}
	for _, p := range files {
		f.write(p, p)
	}
	user, err := f.database.Queries.CreateUser(f.ctx, db.CreateUserParams{Username: "founder", PasswordHash: "h", RecoveryPhraseHash: "r"})
	if err != nil {
		t.Fatal(err)
	}
	f.owner = user.ID
	f.principal = &accessutil.Principal{UserID: user.ID, IsAdmin: true}

	deps := deputil.NewDependencies().
		WithStorageService(storage).
		WithVFSRegistry(registry).
		WithDatabase(f.database)
	gin.SetMode(gin.TestMode)
	f.engine = gin.New()
	f.engine.Use(func(c *gin.Context) {
		c = ctxutil.With(c, "deps", deps)
		c = ctxutil.With(c, "principal", *f.principal)
		c.Next()
	})
	serverutil.RegisterRouterWithGroup(f.engine.Group("/api/v0"), v0_albums.NewRouter())
	return f
}

// write puts content at p on the internal drive.
func (f *pointerFixture) write(p, content string) {
	f.t.Helper()
	if err := f.fsys.Write(f.ctx, p, strings.NewReader(content), vfs.WriteOptions{}); err != nil {
		f.t.Fatalf("write %q: %v", p, err)
	}
}

// read returns the content at p, or fails the test when it is not there.
func (f *pointerFixture) read(p string) string {
	f.t.Helper()
	file, err := f.fsys.Open(f.ctx, p)
	if err != nil {
		f.t.Fatalf("open %q: %v", p, err)
	}
	defer file.Close()
	b, err := io.ReadAll(file)
	if err != nil {
		f.t.Fatalf("read %q: %v", p, err)
	}
	return string(b)
}

// exists reports whether p is a file on the internal drive.
func (f *pointerFixture) exists(p string) bool {
	_, err := f.fsys.Stat(f.ctx, p)
	return err == nil
}

// album creates one of the owner's albums, under parent when it is non-zero.
func (f *pointerFixture) album(name string, parent int64) int64 {
	f.t.Helper()
	created, err := albumutil.CreateAlbum(f.ctx, albumutil.CreateAlbumParams{
		UserID: f.owner, Queries: f.database.Queries, Name: name,
		ParentID: sql.NullInt64{Int64: parent, Valid: parent != 0},
	})
	if err != nil {
		f.t.Fatal(err)
	}
	return created.Album.ID
}

// favorite marks p a favorite of the owner, which puts it in Favorites.
func (f *pointerFixture) favorite(p string) int64 {
	f.t.Helper()
	if _, err := favoritesutil.ToggleFavorite(f.ctx, f.database.Queries, f.owner, "", p); err != nil {
		f.t.Fatal(err)
	}
	fav, err := favoritesutil.EnsureFavoritesAlbum(f.ctx, f.database.Queries, f.owner)
	if err != nil {
		f.t.Fatal(err)
	}
	return fav.ID
}

func (f *pointerFixture) do(method, path, body string, out any) int {
	f.t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	f.engine.ServeHTTP(w, req)
	if out != nil && w.Code < 300 {
		if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
			f.t.Fatalf("%s %s: decode %s: %v", method, path, w.Body.String(), err)
		}
	}
	return w.Code
}

func albumPath(id int64) string { return "/api/v0/albums/" + strconv.FormatInt(id, 10) }

func photoBody(p string) string { return `{"relPath":"` + p + `"}` }

// add puts p in the album over the API.
func (f *pointerFixture) add(album int64, p string) {
	f.t.Helper()
	if code := f.do(http.MethodPost, albumPath(album)+"/items", photoBody(p), nil); code != http.StatusCreated {
		f.t.Fatalf("add %q to album %d = %d, want 201", p, album, code)
	}
}

// items lists the album's paths over the API with the query appended, and
// fails when one names a file that is not on disk: the client would open it
// as a black tile.
func (f *pointerFixture) items(album int64, query string) []string {
	f.t.Helper()
	var list []v0_albums.AlbumItemJSON
	if code := f.do(http.MethodGet, albumPath(album)+"/items"+query, "", &list); code != http.StatusOK {
		f.t.Fatalf("list album %d = %d, want 200", album, code)
	}
	paths := make([]string, 0, len(list))
	for _, item := range list {
		if !f.exists(item.RelPath) {
			f.t.Errorf("album %d lists %q, which is not on disk", album, item.RelPath)
		}
		paths = append(paths, item.RelPath)
	}
	return paths
}

// counts reads each album's itemCount from the album list, by id.
func (f *pointerFixture) counts() map[int64]int64 {
	f.t.Helper()
	var albums []v0_albums.AlbumJSON
	if code := f.do(http.MethodGet, "/api/v0/albums", "", &albums); code != http.StatusOK {
		f.t.Fatalf("list albums = %d, want 200", code)
	}
	got := map[int64]int64{}
	var walk func([]v0_albums.AlbumJSON)
	walk = func(list []v0_albums.AlbumJSON) {
		for _, a := range list {
			got[a.ID] = a.ItemCount
			walk(a.Children)
		}
	}
	walk(albums)
	return got
}

// expect checks an album's items, sorted by name, and that both counts the
// API reports for it agree with them.
func (f *pointerFixture) expect(album int64, want ...string) {
	f.t.Helper()
	if want == nil {
		want = []string{}
	}
	if got := f.items(album, "?sort=name&order=asc"); !reflect.DeepEqual(got, want) {
		f.t.Errorf("album %d items = %v, want %v", album, got, want)
	}
	if got := f.counts()[album]; got != int64(len(want)) {
		f.t.Errorf("album %d itemCount in the list = %d, want %d", album, got, len(want))
	}
	var one v0_albums.AlbumJSON
	f.do(http.MethodGet, albumPath(album), "", &one)
	if one.ItemCount != int64(len(want)) {
		f.t.Errorf("album %d itemCount from get = %d, want %d", album, one.ItemCount, len(want))
	}
}

// deleteFiles trashes paths the way the Files delete route does and waits
// for the database cleanup it starts in the background to finish. Dropping
// photo hashes is that cleanup's last step for each path, so a hash row
// planted at every path is gone once nothing else is left to run.
func (f *pointerFixture) deleteFiles(paths ...string) {
	f.t.Helper()
	for _, p := range paths {
		if err := f.database.Queries.UpsertPhotoHash(f.ctx, db.UpsertPhotoHashParams{
			RelPath: p,
			Dhash:   sql.NullString{String: "0000000000000000", Valid: true},
		}); err != nil {
			f.t.Fatal(err)
		}
	}
	if _, err := fileutil.DeleteFiles(fileutil.DeleteFilesParams{
		Registry:  f.registry,
		EventBus:  eventbus.New(),
		Database:  f.database,
		RootDir:   "/",
		FilePaths: paths,
		TrashedBy: f.owner,
	}); err != nil {
		f.t.Fatalf("DeleteFiles(%v): %v", paths, err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		states, err := f.database.Queries.ListPhotoHashStates(f.ctx)
		if err != nil {
			f.t.Fatal(err)
		}
		if len(states) == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	f.t.Fatalf("the cleanup after deleting %v had not finished after five seconds", paths)
}

// move moves or renames a file the way the Files move route does.
func (f *pointerFixture) move(from, to string) {
	f.t.Helper()
	if _, err := fileutil.MoveFile(fileutil.MoveFileParams{
		Ctx: f.ctx, Registry: f.registry, EventBus: eventbus.New(), Database: f.database,
		OldFilePath: from, NewFilePath: to,
	}); err != nil {
		f.t.Fatalf("MoveFile(%q, %q): %v", from, to, err)
	}
}

// TestAlbumsPointerModel_RemoveKeepsThePhoto: removing a photo from an album
// drops the pointer and nothing else. The count moves with each add and
// remove without anything else happening in between, and the file, its other
// album and its favorite are all still there (#3488).
func TestAlbumsPointerModel_RemoveKeepsThePhoto(t *testing.T) {
	f := newPointerFixture(t, "a.jpg", "b.jpg")
	trip := f.album("trip", 0)
	best := f.album("best", 0)
	fav := f.favorite("a.jpg")

	f.add(trip, "a.jpg")
	f.expect(trip, "a.jpg")
	f.add(trip, "b.jpg")
	f.expect(trip, "a.jpg", "b.jpg")
	f.add(best, "a.jpg")

	if code := f.do(http.MethodDelete, albumPath(trip)+"/items", photoBody("a.jpg"), nil); code != http.StatusNoContent {
		t.Fatalf("remove = %d, want 204", code)
	}
	f.expect(trip, "b.jpg")
	f.expect(best, "a.jpg")
	f.expect(fav, "a.jpg")
	if got := f.read("a.jpg"); got != "a.jpg" {
		t.Errorf("a.jpg after removing it from an album = %q, want its content", got)
	}
}

// TestAlbumsPointerModel_DeleteOriginalDropsEveryPointer: deleting a photo
// that sits in two albums and Favorites removes it from all three, so no
// album is left pointing at a file that is gone, and every count drops with
// it. The neighbor it shared the albums with stays (#3488).
func TestAlbumsPointerModel_DeleteOriginalDropsEveryPointer(t *testing.T) {
	f := newPointerFixture(t, "a.jpg", "b.jpg")
	trip := f.album("trip", 0)
	best := f.album("best", 0)
	fav := f.favorite("a.jpg")
	f.favorite("b.jpg")
	for _, album := range []int64{trip, best} {
		f.add(album, "a.jpg")
		f.add(album, "b.jpg")
	}

	f.deleteFiles("a.jpg")

	if f.exists("a.jpg") {
		t.Fatal("a.jpg is still on disk after the delete")
	}
	for _, album := range []int64{trip, best, fav} {
		f.expect(album, "b.jpg")
	}
}

// TestAlbumsPointerModel_DeleteFolderDropsItsPhotos: deleting a folder drops
// the album pointers to everything under it, and only under it: a sibling
// folder whose name starts the same keeps its photos (#3488).
func TestAlbumsPointerModel_DeleteFolderDropsItsPhotos(t *testing.T) {
	f := newPointerFixture(t, "trip/a.jpg", "trip/day/b.jpg", "trips/c.jpg")
	album := f.album("trip", 0)
	for _, p := range []string{"trip/a.jpg", "trip/day/b.jpg", "trips/c.jpg"} {
		f.add(album, p)
	}

	f.deleteFiles("trip")

	f.expect(album, "trips/c.jpg")
}

// TestAlbumsPointerModel_MoveOriginalFollowsTheFile: renaming or moving a
// photo carries its album pointer to the new path. Neither a same-named photo
// in another folder nor a new file written at the old path is ever shown in
// its place (#3488).
func TestAlbumsPointerModel_MoveOriginalFollowsTheFile(t *testing.T) {
	f := newPointerFixture(t, "trip/a.jpg", "other/a.jpg")
	album := f.album("trip", 0)
	fav := f.favorite("trip/a.jpg")
	f.add(album, "trip/a.jpg")

	f.move("trip/a.jpg", "trip/renamed.jpg")
	f.expect(album, "trip/renamed.jpg")

	f.move("trip/renamed.jpg", "2024/renamed.jpg")
	f.expect(album, "2024/renamed.jpg")
	f.expect(fav, "2024/renamed.jpg")

	f.move("2024", "archive/2024")
	f.expect(album, "archive/2024/renamed.jpg")

	f.write("trip/a.jpg", "a stranger at the old path")
	f.expect(album, "archive/2024/renamed.jpg")
	if got := f.read("archive/2024/renamed.jpg"); got != "trip/a.jpg" {
		t.Errorf("the album's photo reads %q, want the original's content", got)
	}
}

// TestAlbumsPointerModel_CopyIsIndependent: CopyPhoto writes a real file
// beside the original rather than another pointer to it. The album that holds
// the original does not gain the copy, and deleting either one leaves the
// other readable (#3488).
func TestAlbumsPointerModel_CopyIsIndependent(t *testing.T) {
	f := newPointerFixture(t, "a.jpg", "b.jpg")
	album := f.album("trip", 0)
	f.add(album, "a.jpg")
	f.add(album, "b.jpg")

	copyOf := func(p string) string {
		t.Helper()
		copied, err := photoutil.CopyPhoto(photoutil.CopyPhotoParams{Ctx: f.ctx, Registry: f.registry, RelPath: p})
		if err != nil {
			t.Fatalf("CopyPhoto(%q): %v", p, err)
		}
		return copied.RelPath
	}
	aCopy, bCopy := copyOf("a.jpg"), copyOf("b.jpg")
	if aCopy != "a_copy.jpg" {
		t.Errorf("copy of a.jpg = %q, want a_copy.jpg", aCopy)
	}
	f.expect(album, "a.jpg", "b.jpg")

	f.deleteFiles("a.jpg")
	f.expect(album, "b.jpg")
	if got := f.read(aCopy); got != "a.jpg" {
		t.Errorf("%s after deleting its original = %q, want the original's content", aCopy, got)
	}

	f.deleteFiles(bCopy)
	f.expect(album, "b.jpg")
	if got := f.read("b.jpg"); got != "b.jpg" {
		t.Errorf("b.jpg after deleting its copy = %q, want its content", got)
	}
}

// TestAlbumsPointerModel_CoverFallsBackToARealPhoto: the client draws an
// album from the first item it lists. The listing follows the requested sort
// either way, and deleting the photo that came first leaves the next real one
// at the front rather than a pointer to nothing (#3488).
func TestAlbumsPointerModel_CoverFallsBackToARealPhoto(t *testing.T) {
	f := newPointerFixture(t, "a.jpg", "b.jpg", "c.jpg")
	album := f.album("trip", 0)
	for _, p := range []string{"b.jpg", "a.jpg", "c.jpg"} {
		f.add(album, p)
	}

	if got, want := f.items(album, "?sort=name&order=asc"), []string{"a.jpg", "b.jpg", "c.jpg"}; !reflect.DeepEqual(got, want) {
		t.Errorf("by name ascending = %v, want %v", got, want)
	}
	if got, want := f.items(album, "?sort=name&order=desc"), []string{"c.jpg", "b.jpg", "a.jpg"}; !reflect.DeepEqual(got, want) {
		t.Errorf("by name descending = %v, want %v", got, want)
	}

	f.deleteFiles("a.jpg")
	if got := f.items(album, "?sort=name&order=asc"); len(got) == 0 || got[0] != "b.jpg" {
		t.Errorf("first item after deleting the first = %v, want b.jpg first", got)
	}
	f.deleteFiles("b.jpg", "c.jpg")
	f.expect(album)
}

// TestAlbumsPointerModel_DeletingAParentAlbumKeepsThePhotos: deleting an album
// deletes its sub-albums with it, and drops their pointers, but never a photo
// (#3488).
func TestAlbumsPointerModel_DeletingAParentAlbumKeepsThePhotos(t *testing.T) {
	f := newPointerFixture(t, "a.jpg", "b.jpg")
	parent := f.album("trip", 0)
	child := f.album("day1", parent)
	keep := f.album("best", 0)
	f.add(parent, "a.jpg")
	f.add(child, "b.jpg")
	f.add(keep, "b.jpg")

	if code := f.do(http.MethodDelete, albumPath(parent), "", nil); code != http.StatusNoContent {
		t.Fatalf("delete parent album = %d, want 204", code)
	}
	if code := f.do(http.MethodGet, albumPath(child)+"/items", "", nil); code != http.StatusNotFound {
		t.Errorf("list the deleted sub-album = %d, want 404", code)
	}
	if _, ok := f.counts()[child]; ok {
		t.Error("the sub-album is still listed after its parent was deleted")
	}
	f.expect(keep, "b.jpg")
	for _, p := range []string{"a.jpg", "b.jpg"} {
		if !f.exists(p) {
			t.Errorf("%s is gone after deleting the albums that held it", p)
		}
	}
}

// TestAlbumsPointerModel_OtherAccountCannotReachTheAlbum pins today's model:
// an album belongs to the account that made it, and another account, admin or
// not, can neither read its items nor change them. Sharing an album with
// another account is #1952, and whether an admin may read someone else's is
// #3437; when either is decided this test changes with it. TestAlbums_Access
// covers what a non-admin owner sees of their own album.
func TestAlbumsPointerModel_OtherAccountCannotReachTheAlbum(t *testing.T) {
	f := newPointerFixture(t, "a.jpg")
	album := f.album("trip", 0)
	f.add(album, "a.jpg")
	owner := *f.principal

	other, err := f.database.Queries.CreateUser(f.ctx, db.CreateUserParams{Username: "other", PasswordHash: "h", RecoveryPhraseHash: "r"})
	if err != nil {
		t.Fatal(err)
	}
	for _, principal := range []accessutil.Principal{
		{UserID: other.ID},
		{UserID: other.ID, IsAdmin: true},
	} {
		*f.principal = principal
		for _, req := range []struct{ method, path, body string }{
			{http.MethodGet, albumPath(album), ""},
			{http.MethodGet, albumPath(album) + "/items", ""},
			{http.MethodPost, albumPath(album) + "/items", photoBody("a.jpg")},
			{http.MethodDelete, albumPath(album) + "/items", photoBody("a.jpg")},
		} {
			if code := f.do(req.method, req.path, req.body, nil); code != http.StatusNotFound {
				t.Errorf("admin=%v %s %s = %d, want 404", principal.IsAdmin, req.method, req.path, code)
			}
		}
		if _, ok := f.counts()[album]; ok {
			t.Errorf("admin=%v: another account's album is in the list", principal.IsAdmin)
		}
	}

	*f.principal = owner
	f.expect(album, "a.jpg")
}
