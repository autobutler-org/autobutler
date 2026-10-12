package searchutil

import (
	"context"
	"errors"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
)

// On PostgreSQL the index writes are ordinary queries on file_content and
// work, so the content is there when search arrives (#2958). The search
// itself is FTS5: it is refused by name, and no FTS5 SQL reaches the server.
func TestContentSearchOnPostgres(t *testing.T) {
	database, err := db.ConnectToPostgres(dbtest.PostgresURL(t))
	if err != nil {
		t.Fatalf("ConnectToPostgres: %v", err)
	}
	t.Cleanup(func() { database.Db.Close() })
	ctx := context.Background()

	for range 2 {
		if err := UpsertContent(ctx, database.Db, "SER1", "notes/a.txt", "hello world"); err != nil {
			t.Fatalf("UpsertContent: %v", err)
		}
	}
	if err := UpsertContent(ctx, database.Db, "SER2", "b.txt", "other"); err != nil {
		t.Fatalf("UpsertContent: %v", err)
	}
	count := func() int {
		t.Helper()
		var n int
		if err := database.Db.QueryRow(`SELECT COUNT(*) FROM file_content`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if got := count(); got != 2 {
		t.Fatalf("file_content holds %d rows after indexing two files, one of them twice; want 2", got)
	}

	_, err = SearchReadable(SearchReadableParams{Ctx: ctx, DB: database.Db, Query: "hello"})
	var unsupported *db.UnsupportedError
	if !errors.As(err, &unsupported) || unsupported.Issue != 2958 {
		t.Fatalf("SearchReadable = %v, want a *db.UnsupportedError naming #2958", err)
	}
	if want := "Content search is not supported on the PostgreSQL backend yet (#2958)"; err.Error() != want {
		t.Errorf("the refusal reads %q, want %q", err, want)
	}

	if err := DeleteContent(ctx, database.Db, "SER1", "notes/a.txt"); err != nil {
		t.Fatalf("DeleteContent: %v", err)
	}
	if err := DeleteContentBySerial(ctx, database.Db, "SER2"); err != nil {
		t.Fatalf("DeleteContentBySerial: %v", err)
	}
	if got := count(); got != 0 {
		t.Errorf("file_content holds %d rows after deleting both, want 0", got)
	}
}
