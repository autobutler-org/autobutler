package db_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
)

// writeKey is the write every test here makes inside its transaction.
func writeKey(ctx context.Context, q db.Querier, key string) error {
	return q.UpsertVFSMetadata(ctx, db.UpsertVFSMetadataParams{Namespace: "ns", Path: "p", Key: key, Value: `"v"`})
}

// keys lists what is stored outside any transaction, so it sees only what was
// committed.
func keys(t *testing.T, database *db.DatabaseSqlc) []string {
	t.Helper()
	rows, err := database.Queries.ListVFSMetadata(context.Background(), db.ListVFSMetadataParams{Namespace: "ns", Path: "p"})
	if err != nil {
		t.Fatalf("list metadata: %v", err)
	}
	found := make([]string, 0, len(rows))
	for _, row := range rows {
		found = append(found, row.Key)
	}
	return found
}

func TestInTx_CommitsWhenFnReturnsNil(t *testing.T) {
	database := dbtest.NewDB(t)
	ctx := context.Background()

	err := db.InTx(ctx, database.Db, func(q db.Querier) error {
		return writeKey(ctx, q, "kept")
	})
	if err != nil {
		t.Fatalf("InTx: %v", err)
	}
	if got := keys(t, database); len(got) != 1 || got[0] != "kept" {
		t.Errorf("stored keys = %v, want [kept]", got)
	}
}

func TestInTx_RollsBackAndReturnsFnErrorUnwrapped(t *testing.T) {
	database := dbtest.NewDB(t)
	ctx := context.Background()
	errSentinel := errors.New("sentinel")

	err := db.InTx(ctx, database.Db, func(q db.Querier) error {
		if err := writeKey(ctx, q, "dropped"); err != nil {
			return err
		}
		return errSentinel
	})
	// Identity, not errors.Is: callers get the callback's error itself.
	if err != errSentinel {
		t.Errorf("InTx error = %v, want the callback's own error", err)
	}
	if got := keys(t, database); len(got) != 0 {
		t.Errorf("stored keys = %v, want none after a rollback", got)
	}
}

func TestInTx_RollsBackWhenFnPanics(t *testing.T) {
	database := dbtest.NewDB(t)
	ctx := context.Background()

	func() {
		defer func() {
			if recover() == nil {
				t.Error("the panic did not propagate out of InTx")
			}
		}()
		_ = db.InTx(ctx, database.Db, func(q db.Querier) error {
			if err := writeKey(ctx, q, "dropped"); err != nil {
				t.Fatalf("write: %v", err)
			}
			panic("boom")
		})
	}()

	if got := keys(t, database); len(got) != 0 {
		t.Errorf("stored keys = %v, want none after a panic", got)
	}
	// A transaction left open would still hold the write lock, and this would
	// fail busy.
	if err := writeKey(ctx, database.Queries, "after"); err != nil {
		t.Errorf("write after the panic: %v", err)
	}
}

func TestInTx_WrapsBeginFailure(t *testing.T) {
	database := dbtest.NewDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	called := false
	err := db.InTx(ctx, database.Db, func(db.Querier) error {
		called = true
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("InTx error = %v, want one wrapping context.Canceled", err)
	}
	if called {
		t.Error("the callback ran without a transaction")
	}
}

// A transaction that reads and then writes must not fail with SQLITE_BUSY when
// another connection starts writing in between. SQLite refuses to upgrade a
// deferred transaction's read lock at once — without waiting out busy_timeout —
// when another writer is queued behind it, so POST /vault/items (read the key,
// insert the item) answered 500 whenever a background write landed between the
// two. InTx's BEGIN IMMEDIATE takes the write lock first, where the
// busy timeout applies.
func TestInTx_ReadThenWriteSurvivesAnotherWriter(t *testing.T) {
	database := dbtest.NewDB(t)
	ctx := context.Background()

	otherDone := make(chan error, 1)
	err := db.InTx(ctx, database.Db, func(q db.Querier) error {
		if _, err := q.ListVFSMetadata(ctx, db.ListVFSMetadataParams{Namespace: "ns", Path: "p"}); err != nil {
			return err
		}
		go func() {
			_, err := database.Db.ExecContext(ctx, `INSERT INTO vfs_metadata (namespace, path, key, value) VALUES ('ns', 'p', 'other', '"v"')`)
			otherDone <- err
		}()
		// Let the other writer take its place behind this transaction.
		time.Sleep(200 * time.Millisecond)
		return writeKey(ctx, q, "mine")
	})
	if err != nil {
		t.Fatalf("InTx: %v", err)
	}
	if err := <-otherDone; err != nil {
		t.Fatalf("the other writer: %v", err)
	}
}
