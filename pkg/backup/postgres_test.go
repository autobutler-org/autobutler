package backup

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
)

// postgresDatabase is an empty, migrated PostgreSQL database of the test's
// own, opened the way the server opens one.
func postgresDatabase(t *testing.T) *db.DatabaseSqlc {
	t.Helper()
	database, err := db.ConnectToPostgres(dbtest.PostgresURL(t))
	if err != nil {
		t.Fatalf("ConnectToPostgres: %v", err)
	}
	t.Cleanup(func() { database.Db.Close() })
	return database
}

// expectUnsupported fails unless err is the refusal a feature that is not on
// PostgreSQL yet gives, naming issue.
func expectUnsupported(t *testing.T, what string, err error, issue int) {
	t.Helper()
	var unsupported *db.UnsupportedError
	if !errors.As(err, &unsupported) || unsupported.Issue != issue {
		t.Errorf("%s = %v, want a *db.UnsupportedError naming #%d", what, err, issue)
	}
}

// The chat backup attaches a SQLite file to the live connection. On
// PostgreSQL it says so and writes nothing, and a snapshot, which always
// carries one, is refused before anything is copied.
func TestChatBackupOnPostgresIsUnsupported(t *testing.T) {
	database := postgresDatabase(t)
	ctx := context.Background()
	dir := t.TempDir()

	_, err := ExportChat(ctx, database.Db, dir)
	expectUnsupported(t, "ExportChat", err, 2963)
	_, err = ImportChat(ctx, database.Db, dir)
	expectUnsupported(t, "ImportChat", err, 2963)
	_, err = ExportVault(ctx, database.Db, dir)
	expectUnsupported(t, "ExportVault", err, 2963)
	_, err = ImportVault(ctx, database.Db, dir)
	expectUnsupported(t, "ImportVault", err, 2963)
	if left, err := os.ReadDir(dir); err != nil || len(left) != 0 {
		t.Errorf("the refused backups left %v in the target directory (%v)", left, err)
	}

	_, err = StartSnapshotBackup(StartSnapshotBackupParams{
		Ctx: ctx, Queries: database.Queries, LiveDB: database.Db, TargetDeviceSerial: "SER1",
	})
	expectUnsupported(t, "StartSnapshotBackup", err, 2963)
	// The same call on SQLite gets past that check, to the next refusal.
	sqlite := dbtest.NewDB(t)
	if _, err := StartSnapshotBackup(StartSnapshotBackupParams{
		Ctx: ctx, Queries: sqlite.Queries, LiveDB: sqlite.Db, TargetDeviceSerial: "SER1",
	}); !errors.Is(err, ErrTargetRoleRequired) {
		t.Errorf("StartSnapshotBackup on SQLite = %v, want ErrTargetRoleRequired", err)
	}
}
