package musicutil_test

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
	"github.com/autobutler-org/quark/pkg/util/musicutil"
)

// TestMusicOnPostgres runs the scan and every browse query on PostgreSQL,
// where they are shared sqlc queries, and checks they agree with SQLite.
// Search is FTS5: it is refused by name until #2958.
func TestMusicOnPostgres(t *testing.T) {
	database, err := db.ConnectToPostgres(dbtest.PostgresURL(t))
	if err != nil {
		t.Fatalf("ConnectToPostgres: %v", err)
	}
	t.Cleanup(func() { _ = database.Db.Close() })
	sqlite := library(t)
	ctx := context.Background()
	seed(t, database.Db, append(slices.Clone(libraryTracks), mixedCaseTracks...)...)
	seed(t, sqlite.Db, mixedCaseTracks...)
	admin := adminAccess(t)

	for _, sort := range []musicutil.TrackSort{musicutil.SortAlbumOrder, musicutil.SortTitle, musicutil.SortYear, musicutil.SortAdded} {
		for _, desc := range []bool{false, true} {
			params := musicutil.ListTracksParams{Ctx: ctx, Access: admin, Sort: sort, Descending: desc}
			params.DB = sqlite.Db
			want, err := musicutil.ListTracks(params)
			if err != nil {
				t.Fatal(err)
			}
			params.DB = database.Db
			got, err := musicutil.ListTracks(params)
			if err != nil {
				t.Fatalf("ListTracks sort %q on PostgreSQL: %v", sort, err)
			}
			if !reflect.DeepEqual(titles(got.Tracks), titles(want.Tracks)) || got.Total != want.Total {
				t.Errorf("sort %q desc %v: PostgreSQL %v, SQLite %v", sort, desc, titles(got.Tracks), titles(want.Tracks))
			}
		}
	}
	// Displayed names are compared too: a group's spelling must not follow
	// the engine's collation (#3306).
	for name, list := range map[string]func(*sql.DB) (any, error){
		"albums": func(conn *sql.DB) (any, error) {
			return musicutil.ListAlbums(musicutil.ListAlbumsParams{Ctx: ctx, DB: conn, Access: admin})
		},
		"artists": func(conn *sql.DB) (any, error) {
			return musicutil.ListArtists(musicutil.ListArtistsParams{Ctx: ctx, DB: conn, Access: admin})
		},
		"genres": func(conn *sql.DB) (any, error) {
			return musicutil.ListGenres(musicutil.ListGenresParams{Ctx: ctx, DB: conn, Access: admin})
		},
	} {
		want, err := list(sqlite.Db)
		if err != nil {
			t.Fatal(err)
		}
		got, err := list(database.Db)
		if err != nil {
			t.Errorf("%s on PostgreSQL: %v", name, err)
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: PostgreSQL %+v\nSQLite     %+v", name, got, want)
		}
	}
	if _, err := musicutil.MovePath(musicutil.MovePathParams{Ctx: ctx, DB: database.Db, Serial: usbSerial, OldPath: "shared", NewPath: "moved"}); err != nil {
		t.Errorf("MovePath: %v", err)
	}

	_, err = musicutil.Search(musicutil.SearchParams{Ctx: ctx, DB: database.Db, Access: admin, Query: "ann"})
	var unsupported *db.UnsupportedError
	if !errors.As(err, &unsupported) || unsupported.Issue != 2958 {
		t.Errorf("Search = %v, want a *db.UnsupportedError naming #2958", err)
	}
}
