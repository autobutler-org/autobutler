package backup

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
	"github.com/autobutler-org/quark/pkg/util/vaultcrypto"
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
	if left, err := os.ReadDir(dir); err != nil || len(left) != 0 {
		t.Errorf("the refused chat backup left %v in the target directory (%v)", left, err)
	}

	_, err = StartSnapshotBackup(StartSnapshotBackupParams{
		Ctx: ctx, Queries: database.Queries, ChatDB: database.Db, TargetDeviceSerial: "SER1",
	})
	expectUnsupported(t, "StartSnapshotBackup", err, 2963)
	// The same call on SQLite gets past that check, to the next refusal.
	sqlite := dbtest.NewDB(t)
	if _, err := StartSnapshotBackup(StartSnapshotBackupParams{
		Ctx: ctx, Queries: sqlite.Queries, ChatDB: sqlite.Db, TargetDeviceSerial: "SER1",
	}); !errors.Is(err, ErrTargetRoleRequired) {
		t.Errorf("StartSnapshotBackup on SQLite = %v, want ErrTargetRoleRequired", err)
	}
}

// Moving the vault copies rows with SQLite's placeholders. With PostgreSQL on
// either side it is refused and neither side changes; emptying a vault is
// plain SQL and works on both.
func TestVaultMoveOnPostgresIsUnsupported(t *testing.T) {
	postgres := postgresDatabase(t)
	ctx := context.Background()
	sqlite := dbtest.NewDB(t)
	if _, err := sqlite.Queries.CreateVaultFolder(ctx, db.CreateVaultFolderParams{Name: "Social"}); err != nil {
		t.Fatal(err)
	}
	if _, err := postgres.Queries.CreateVaultFolder(ctx, db.CreateVaultFolderParams{Name: "Work"}); err != nil {
		t.Fatal(err)
	}

	expectUnsupported(t, "MigrateVault from SQLite to PostgreSQL", MigrateVault(ctx, sqlite, postgres), 2963)
	expectUnsupported(t, "MigrateVault from PostgreSQL to SQLite", MigrateVault(ctx, postgres, sqlite), 2963)
	for name, database := range map[string]*db.DatabaseSqlc{"SQLite": sqlite, "PostgreSQL": postgres} {
		if folders, err := database.Queries.ListVaultFolders(ctx); err != nil || len(folders) != 1 {
			t.Errorf("the refused move left %d folders on %s (%v), want the 1 it had", len(folders), name, err)
		}
	}

	if err := TruncateVaultTables(ctx, postgres); err != nil {
		t.Fatalf("TruncateVaultTables on PostgreSQL: %v", err)
	}
	if folders, err := postgres.Queries.ListVaultFolders(ctx); err != nil || len(folders) != 0 {
		t.Errorf("TruncateVaultTables left %d folders on PostgreSQL (%v)", len(folders), err)
	}
}

// A vault backup is always a SQLite file. With PostgreSQL as the live
// database the export still writes one, carrying each row's times across, and
// the import still restores it.
func TestVaultBackupRoundTripsOnPostgres(t *testing.T) {
	ctx := context.Background()
	// newVault gives an empty PostgreSQL database a vault and returns its key.
	newVault := func(database *db.DatabaseSqlc, masterPassword string) []byte {
		params := vaultcrypto.Argon2Params{Memory: 1024, Iterations: 1, Parallelism: 1}
		salt, _ := vaultcrypto.GenerateSalt()
		key := vaultcrypto.DeriveKey(masterPassword, salt, params)
		blob, nonce, _ := vaultcrypto.MakeVerificationBlob(key)
		if err := database.Queries.CreateVaultConfig(ctx, db.CreateVaultConfigParams{
			Salt: salt, Argon2Memory: int64(params.Memory), Argon2Iterations: int64(params.Iterations),
			Argon2Parallelism: int64(params.Parallelism), VerificationBlob: blob, VerificationNonce: nonce,
			AutoLockSeconds: 900,
		}); err != nil {
			t.Fatal(err)
		}
		return key
	}

	live := postgresDatabase(t)
	liveKey := newVault(live, "master-pw")
	folder, err := live.Queries.CreateVaultFolder(ctx, db.CreateVaultFolderParams{Name: "Social"})
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, nonce, _ := vaultcrypto.Encrypt(liveKey, []byte(`{"username":"alice","password":"gh-pass"}`))
	entry, err := live.Queries.CreateVaultEntry(ctx, db.CreateVaultEntryParams{
		Name: "GitHub", UrlHost: "github.com", FolderID: sql.NullInt64{Int64: folder.ID, Valid: true},
		Ciphertext: ciphertext, Nonce: nonce,
	})
	if err != nil {
		t.Fatal(err)
	}

	backupDir := t.TempDir()
	path, err := ExportVault(ctx, live.Queries, liveKey, "recovery-pw", backupDir)
	if err != nil {
		t.Fatalf("ExportVault from PostgreSQL: %v", err)
	}
	// The file is SQLite, and holds the entry's times to the second it keeps.
	file, err := sql.Open("sqlite", db.DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var createdAt, updatedAt time.Time
	if err := file.QueryRow(`SELECT created_at, updated_at FROM vault_entries WHERE id = ?`, entry.ID).Scan(&createdAt, &updatedAt); err != nil {
		t.Fatalf("read the exported entry: %v", err)
	}
	if !createdAt.Equal(entry.CreatedAt.Truncate(time.Second)) || !updatedAt.Equal(entry.UpdatedAt.Truncate(time.Second)) {
		t.Errorf("the export holds created_at %v and updated_at %v, want %v and %v to the second",
			createdAt, updatedAt, entry.CreatedAt, entry.UpdatedAt)
	}

	fresh := postgresDatabase(t)
	freshKey := newVault(fresh, "new-master")
	var imported *VaultImportResult
	if err := db.InTx(ctx, fresh.Db, func(tx db.Querier) error {
		var err error
		imported, err = ImportVault(ctx, tx, freshKey, "recovery-pw", backupDir)
		return err
	}); err != nil {
		t.Fatalf("ImportVault into PostgreSQL: %v", err)
	}
	if imported.EntriesImported != 1 || imported.FoldersImported != 1 {
		t.Fatalf("imported %d entries and %d folders, want 1 and 1", imported.EntriesImported, imported.FoldersImported)
	}
	restored, err := fresh.Queries.ListAllVaultEntriesForReEncrypt(ctx)
	if err != nil || len(restored) != 1 {
		t.Fatalf("the restored vault holds %d entries (%v), want 1", len(restored), err)
	}
	plaintext, err := vaultcrypto.Decrypt(freshKey, restored[0].Ciphertext, restored[0].Nonce)
	if err != nil || string(plaintext) != `{"username":"alice","password":"gh-pass"}` {
		t.Errorf("the restored entry decrypts to %q (%v)", plaintext, err)
	}
}
