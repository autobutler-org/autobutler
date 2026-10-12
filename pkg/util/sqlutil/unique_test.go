package sqlutil_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
	"github.com/autobutler-org/quark/pkg/util/sqlutil"
)

// sqlStateErr stands in for the PostgreSQL driver's error: all
// IsUniqueConstraintErr reads of one is its SQLSTATE.
type sqlStateErr string

func (e sqlStateErr) Error() string    { return "ERROR: (SQLSTATE " + string(e) + ")" }
func (e sqlStateErr) SQLState() string { return string(e) }

// What each engine's driver really returns: a second account of one name is a
// unique violation on both, and an album naming an account that is not there,
// a foreign key violation, is one on neither.
func TestIsUniqueConstraintErrOnEachEngine(t *testing.T) {
	check := func(t *testing.T, database *db.DatabaseSqlc) {
		t.Helper()
		ctx := context.Background()
		if _, err := database.Queries.CreateUser(ctx, db.CreateUserParams{Username: "ada"}); err != nil {
			t.Fatal(err)
		}
		_, err := database.Queries.CreateUser(ctx, db.CreateUserParams{Username: "ada"})
		if err == nil || !sqlutil.IsUniqueConstraintErr(err) {
			t.Errorf("a second account named ada gave %v, which IsUniqueConstraintErr does not recognize", err)
		}
		if !sqlutil.IsUniqueConstraintErr(fmt.Errorf("create account: %w", err)) {
			t.Errorf("IsUniqueConstraintErr lost %v once it was wrapped", err)
		}
		// The same violation inside a transaction comes out of db.InTx as it is.
		err = db.InTx(ctx, database.Db, func(tx db.Querier) error {
			_, err := tx.CreateUser(ctx, db.CreateUserParams{Username: "ada"})
			return err
		})
		if !sqlutil.IsUniqueConstraintErr(err) {
			t.Errorf("a second account named ada inside db.InTx gave %v", err)
		}
		_, err = database.Queries.CreateAlbum(ctx, db.CreateAlbumParams{Name: "Summer", UserID: 9999})
		if err == nil || sqlutil.IsUniqueConstraintErr(err) {
			t.Errorf("an album for an account that is not there gave %v, want an error that is not a unique violation", err)
		}
	}
	t.Run("sqlite", func(t *testing.T) { check(t, dbtest.NewDB(t)) })
	t.Run("postgres", func(t *testing.T) {
		database, err := db.ConnectToPostgres(dbtest.PostgresURL(t))
		if err != nil {
			t.Fatalf("ConnectToPostgres: %v", err)
		}
		t.Cleanup(func() { database.Db.Close() })
		check(t, database)
	})
}

func TestIsUniqueConstraintErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil is not a constraint error", nil, false},
		{"ErrNoRows is not a constraint error", sql.ErrNoRows, false},
		{"unrelated error", errors.New("connection refused"), false},
		{"bare UNIQUE constraint failed", errors.New("UNIQUE constraint failed"), true},
		{"driver-prefixed constraint error",
			errors.New("constraint failed: UNIQUE constraint failed: photo_albums.name"), true},
		{"expression index constraint error",
			errors.New("constraint failed: UNIQUE constraint failed: index 'idx_photo_albums_sibling_name'"), true},
		{"wrapped constraint error",
			fmt.Errorf("ensure album: %w", errors.New("UNIQUE constraint failed: photo_albums.name")), true},
		{"lowercase does not match", errors.New("unique constraint failed"), false},
		{"other constraint kinds do not match",
			errors.New("NOT NULL constraint failed: photo_albums.name"), false},
		{"FOREIGN KEY constraint does not match",
			errors.New("FOREIGN KEY constraint failed"), false},
		{"PostgreSQL unique_violation", sqlStateErr("23505"), true},
		{"wrapped PostgreSQL unique_violation", fmt.Errorf("create account: %w", sqlStateErr("23505")), true},
		{"PostgreSQL foreign_key_violation", sqlStateErr("23503"), false},
		{"PostgreSQL not_null_violation", sqlStateErr("23502"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sqlutil.IsUniqueConstraintErr(tt.err); got != tt.want {
				t.Errorf("IsUniqueConstraintErr(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
