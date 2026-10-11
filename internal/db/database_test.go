package db

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

// newTxDB is a migrated database with InTx's callback writing one vault folder
// to it, and a count of the folders that stayed.
func newTxDB(t *testing.T) (database *DatabaseSqlc, addFolder func(*Queries) error, folders func() int) {
	t.Helper()
	database = openInstance(t, filepath.Join(t.TempDir(), "quark.db"))
	if err := ResetDatabase(database); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	addFolder = func(q *Queries) error {
		_, err := q.CreateVaultFolder(context.Background(), CreateVaultFolderParams{Name: "kept"})
		return err
	}
	folders = func() int {
		t.Helper()
		rows, err := database.Queries.ListVaultFolders(context.Background())
		if err != nil {
			t.Fatalf("list folders: %v", err)
		}
		return len(rows)
	}
	return database, addFolder, folders
}

func TestInTx_CommitsWhenTheCallbackSucceeds(t *testing.T) {
	database, addFolder, folders := newTxDB(t)

	if err := InTx(context.Background(), database.Db, addFolder); err != nil {
		t.Fatalf("InTx: %v", err)
	}
	if got := folders(); got != 1 {
		t.Errorf("folders after commit = %d, want 1", got)
	}
}

func TestInTx_RollsBackAndReturnsTheCallbackError(t *testing.T) {
	database, addFolder, folders := newTxDB(t)
	errStop := errors.New("stop")

	err := InTx(context.Background(), database.Db, func(q *Queries) error {
		if err := addFolder(q); err != nil {
			return err
		}
		return errStop
	})
	if err != errStop {
		t.Errorf("InTx error = %v, want the callback's own error", err)
	}
	if got := folders(); got != 0 {
		t.Errorf("folders after rollback = %d, want 0", got)
	}
}

// The four private copies this replaced left the transaction open on its
// connection when the callback panicked (#3121).
func TestInTx_RollsBackAndPanicsAgainWhenTheCallbackPanics(t *testing.T) {
	database, addFolder, folders := newTxDB(t)

	func() {
		defer func() {
			if got := recover(); got != "boom" {
				t.Errorf("recovered %v, want the callback's panic", got)
			}
		}()
		_ = InTx(context.Background(), database.Db, func(q *Queries) error {
			if err := addFolder(q); err != nil {
				t.Fatalf("add folder: %v", err)
			}
			panic("boom")
		})
	}()

	if got := folders(); got != 0 {
		t.Errorf("folders after a panic = %d, want 0", got)
	}
	if open := database.Db.Stats().InUse; open != 0 {
		t.Errorf("connections still held after a panic = %d, want 0", open)
	}
}

func TestInTx_ReportsAFailedBegin(t *testing.T) {
	database, addFolder, _ := newTxDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := InTx(ctx, database.Db, addFolder); !errors.Is(err, context.Canceled) {
		t.Errorf("InTx on a canceled context = %v, want context.Canceled", err)
	}
}
